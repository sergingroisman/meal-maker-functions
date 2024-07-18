FROM golang:1.22-alpine AS builder

WORKDIR /serverless

COPY go.* ./

RUN go mod download

COPY . .

RUN go build -o app cmd/main.go

# Final image based on Azure Functions runtime
FROM mcr.microsoft.com/azure-functions/go:3.0

WORKDIR /home/site/wwwroot

# Copy the compiled binary from the builder stage
COPY --from=builder /serverless/app ./

# Set the entrypoint to execute the function app
ENTRYPOINT ["./app"]

# Document that the service listens on port 3000
EXPOSE 8080