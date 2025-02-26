#  Stage 1: Build Stage
FROM golang:1.24-alpine as builder

WORKDIR /build

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

#  Install tools
RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN go install github.com/go-task/task/v3/cmd/task@latest
RUN go install github.com/air-verse/air@latest  

#  Build binary
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN task build || go build -o http-server ./cmd/http/main.go

FROM golang:1.24-alpine

WORKDIR /app

COPY . .

COPY --from=builder /go/bin/air /usr/local/bin/air  

# Set environment variables
ENV GO_ENV=development
ENV PORT=8080

EXPOSE 8080

CMD ["air"]
