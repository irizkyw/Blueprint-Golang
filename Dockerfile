FROM golang:1.24-alpine as builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest 
RUN go install github.com/go-task/task/v3/cmd/task@latest

# RUN task docs || true

# ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
# RUN task build || go build -o http-server ./cmd/http/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /build/http-server .

ENV GO_ENV=production

EXPOSE 8080

CMD ["./http-server"]
