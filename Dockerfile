FROM golang:1.22-alpine AS builder
WORKDIR /app

RUN go mod init simple-mail-server && go get gopkg.in/yaml.v3

COPY main.go config.yaml ./
RUN CGO_ENABLED=0 GOOS=linux go build -o simple-mail-server main.go

FROM alpine:latest

RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

USER nonroot

WORKDIR /root/
COPY --from=builder /app/simple-mail-server .

EXPOSE 8080
CMD ["./simple-mail-server"]
