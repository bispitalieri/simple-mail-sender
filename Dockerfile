FROM golang:1.27-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o simple-mail-server .

FROM alpine:3

RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

WORKDIR /app

COPY --from=builder /app/simple-mail-server .
COPY --from=builder /app/config.yaml .

USER nonroot

EXPOSE 8080

CMD ["./simple-mail-server"]
