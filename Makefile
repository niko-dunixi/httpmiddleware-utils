test: fmt
	go test -v --count=1 ./...

fmt: tidy
	go fmt ./...

tidy:
	go mod tidy

download:
	go mod download