build:
	@go build handler.go

azure-functions-local:
	@func start

swagger:
	bash swagger.sh

run-dev:
	go run handler.go