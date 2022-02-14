BINDIR=bin

build:
	go build -o "${BINDIR}/layercake" ./cmd/layercake/...
	go build -o "${BINDIR}/stagemaker" ./cmd/stagemaker/...


test:
	go test ./...

