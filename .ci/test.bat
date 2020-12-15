mkdir -p build
SET OUT_FILE=build\output-report.out
go test "./..." -v > %OUT_FILE% | type %OUT_FILE%

go get -v -u github.com/jstemmer/go-junit-report
go-junit-report > build\junit-%GO_VERSION%.xml < %OUT_FILE%
