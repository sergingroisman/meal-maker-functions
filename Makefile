swagger:
	bash scripts/swagger.sh

build:
	go build -o app cmd/main.go

build_linux:
	GOOS=linux GOARCH=amd64 go build -o app cmd/main.go

start:
	func start

dev:
	go run cmd/main.go