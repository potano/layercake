BINDIR=bin

build:
	go build -o "${BINDIR}/layercake" layercake.go


test:
	go test ./...

