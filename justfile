default:
    @just --list

build:
    go build -o goob ./cmd/goob

run *args: build
    ./goob {{ args }}

run-grey: build
    ./goob -manifest assets/cat-grey.json

test:
    go test ./...

vet:
    go vet ./...

fmt:
    go fmt ./...

tidy:
    go mod tidy

clean:
    rm -f goob
