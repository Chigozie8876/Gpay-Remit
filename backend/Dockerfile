FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gpay-remit-api main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app

WORKDIR /app

COPY --from=builder --chown=app:app /app/gpay-remit-api .

EXPOSE 8080

USER app

CMD ["./gpay-remit-api"]
