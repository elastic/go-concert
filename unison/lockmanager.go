// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package unison

import (
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/elastic/go-concert"
	"github.com/elastic/go-concert/atomic"
)

type LockManager struct {
	mu    sync.Mutex
	table map[string]*lockEntry
}

type ManagedLock struct {
	key     string
	manager *LockManager
	session *LockSession
	entry   *lockEntry
}

type LockSession struct {
	isLocked             atomic.Bool
	done, unlocked, lost *sigChannel
}

type lockEntry struct {
	session    *LockSession
	muInternal sync.Mutex // internal mutex

	// shared user mutex
	Mutex

	// book keeping, so we can remove the entry from the lock manager if there
	// are not more references to this entry.
	key string
	ref concert.RefCount
}

type sigChannel struct {
	once sync.Once
	ch   chan struct{}
}

var managedLockFinalizer = (*ManagedLock).finalize

func NewLockManager() *LockManager {
	return &LockManager{table: map[string]*lockEntry{}}
}

func (m *LockManager) Access(key string) *ManagedLock {
	return newManagedLock(m, key)
}

func (m *LockManager) ForceUnlock(key string) {
	m.mu.Lock()
	entry := m.findEntry(key)
	m.mu.Unlock()

	entry.muInternal.Lock()
	session := entry.session
	if session != nil {
		entry.session = nil
		if session.isLocked.Load() {
			entry.Mutex.Unlock()
		}

	}
	entry.muInternal.Unlock()

	session.forceUnlock()
	m.releaseEntry(entry)
}

func (m *LockManager) ForceUnlockAll() {
	m.mu.Lock()
	for _, entry := range m.table {
		entry.muInternal.Lock()
		session := entry.session
		if session != nil && session.isLocked.Load() {
			entry.session = nil
			entry.Mutex.Unlock()
		}
		entry.muInternal.Unlock()

		session.forceUnlock()
	}
	m.mu.Unlock()
}

func (m *LockManager) createEntry(key string) *lockEntry {
	entry := &lockEntry{
		Mutex: MakeMutex(),
		key:   key,
	}
	m.table[key] = entry
	return entry
}

func (m *LockManager) findEntry(key string) *lockEntry {
	entry := m.table[key]
	if entry != nil {
		entry.ref.Retain()
	}
	return entry
}

func (m *LockManager) findOrCreate(key string, create bool) (entry *lockEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry = m.findEntry(key); entry == nil && create {
		entry = m.createEntry(key)
	}
	return entry
}

func (m *LockManager) releaseEntry(entry *lockEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry.ref.Release() {
		delete(m.table, entry.key)
	}
}

func newManagedLock(mngr *LockManager, key string) *ManagedLock {
	ml := &ManagedLock{key: key, manager: mngr}
	return ml
}

func (ml *ManagedLock) finalize() {
	defer ml.unlink()
	if ml.session != nil {
		ml.doUnlock()
	}
}

// Key reports the key the lock will lock/unlock.
func (ml *ManagedLock) Key() string {
	return ml.key
}

// Lock the key. It blocks until the lock becomes
// available.
func (ml *ManagedLock) Lock() *LockSession {
	checkNoActiveLockSession(ml.session)

	ml.link(true)
	ml.entry.Lock()
	return ml.markLocked()
}

func (ml *ManagedLock) TryLock() (*LockSession, bool) {
	checkNoActiveLockSession(ml.session)

	ml.link(true)
	if !ml.entry.TryLock() {
		ml.unlink()
		return nil, false
	}

	return ml.markLocked(), true
}

func (ml *ManagedLock) LockTimeout(duration time.Duration) (*LockSession, bool) {
	checkNoActiveLockSession(ml.session)

	ml.link(true)
	if !ml.entry.LockTimeout(duration) {
		ml.unlink()
		return nil, false
	}
	return ml.markLocked(), ml.IsLocked()
}

func (ml *ManagedLock) LockContext(context doneContext) (*LockSession, error) {
	checkNoActiveLockSession(ml.session)

	ml.link(true)
	err := ml.entry.LockContext(context)
	if err != nil {
		ml.unlink()
		return nil, err
	}

	return ml.markLocked(), nil
}

// Unlock releases a resource.
func (ml *ManagedLock) Unlock() {
	checkActiveLockSession(ml.session)
	ml.doUnlock()
}

func (ml *ManagedLock) doUnlock() {
	session, entry := ml.session, ml.entry

	// The lock can be forcefully and asynchronously unreleased by the
	// LockManager. We can only unlock the entry, iff our mutex session is
	// still locked and the entries lock session still matches our lock session.
	// If none of these is the case, then the session was already closed.
	entry.muInternal.Lock()
	if session == entry.session {
		entry.session = nil
		if session.isLocked.Load() {
			entry.Unlock()
		}
	}
	entry.muInternal.Unlock()

	// always signal unlock, independent of the current state of the registry.
	// This will trigger the 'Unlocked' signal, indicating to session listeners that
	// the routine holding the lock deliberately unlocked the ManagedLock.
	// Note: a managed lock must always be explicitely unlocked, no matter of the
	// session state.
	session.unlock()

	ml.unlink()
	ml.markUnlocked()
}

// IsLocked checks if the resource currently holds the lock for the key
func (ml *ManagedLock) IsLocked() bool {
	return ml.session != nil && ml.session.isLocked.Load()
}

func (ml *ManagedLock) link(create bool) {
	if ml.entry == nil {
		ml.entry = ml.manager.findOrCreate(ml.key, create)
	}
}

func (ml *ManagedLock) unlink() {
	if ml.entry == nil {
		return
	}

	entry := ml.entry
	ml.entry = nil
	ml.manager.releaseEntry(entry)
}

func (ml *ManagedLock) markLocked() *LockSession {
	session := newLockSession()
	ml.session = session

	ml.entry.muInternal.Lock()
	ml.entry.session = session
	ml.entry.muInternal.Unlock()

	// in case we miss an unlock operation (programmer error or panic that hash
	// been caught) we set a finalizer to eventually free the resource.
	// The Unlock operation will unsert the finalizer.
	runtime.SetFinalizer(ml, managedLockFinalizer)
	return session
}

func (ml *ManagedLock) markUnlocked() {
	runtime.SetFinalizer(ml, nil)
}

func newLockSession() *LockSession {
	return &LockSession{
		isLocked: atomic.MakeBool(true),
		done:     newSigChannel(),
		unlocked: newSigChannel(),
		lost:     newSigChannel(),
	}
}

func (s *LockSession) Done() <-chan struct{}     { return s.done.Sig() }
func (s *LockSession) Unlocked() <-chan struct{} { return s.unlocked.Sig() }
func (s *LockSession) LockLost() <-chan struct{} { return s.lost.Sig() }

func (s *LockSession) unlock()      { s.doUnlock(s.unlocked) }
func (s *LockSession) forceUnlock() { s.doUnlock(s.lost) }
func (s *LockSession) doUnlock(kind *sigChannel) {
	s.isLocked.Store(false)
	kind.Close()
	s.done.Close()
}

func newSigChannel() *sigChannel {
	return &sigChannel{ch: make(chan struct{})}
}

func (s *sigChannel) Sig() <-chan struct{} {
	return s.ch
}

func (s *sigChannel) Close() {
	s.once.Do(func() {
		close(s.ch)
	})
}

func checkNoActiveLockSession(s *LockSession) {
	invariant(s == nil, "lock still has an active lock session, missing call to Unlock to finish the session")
}

func checkActiveLockSession(s *LockSession) {
	invariant(s != nil, "no active lock session")
}

func invariant(b bool, message string) {
	if !b {
		panic(errors.New(message))
	}
}
