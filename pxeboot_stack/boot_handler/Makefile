build:
	go mod tidy
	go mod download
	CGO_ENABLED=0 GOOS=linux go build -o bin/boot_handler .

.PHONY: build