build:
	go build
	go test
	go vet

check:
	staticcheck

fmt:
	gofmt -w -s *.go

clean:
	go clean

tidy:
	go mod vendor
	go mod tidy
