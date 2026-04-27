FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod ./
# Si tienes go.sum, descomenta la siguiente línea
# COPY go.sum ./
RUN go mod download
COPY . .
RUN go build -o main ./backend

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
