FROM golang:1.24.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .


RUN CGO_ENABLED=0 GOOS=linux go build -o /bot main.go



FROM alpine:latest
WORKDIR /root
COPY --from=builder /bot /bot

RUN apk --no-cache add ca-certificates

CMD ["/bot"]
