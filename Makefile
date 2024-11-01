build:
	@go build -o bin/cdn
run: build
	@./bin/cdn
test:
	@go test -v ./...