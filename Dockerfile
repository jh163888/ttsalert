FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd

FROM alpine:latest

RUN apk --no-cache add ca-certificates ffmpeg

RUN apk add --no-cache \
    && pip3 install --break-system-packages edge-tts

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/configs/config.example.yaml ./config.yaml

EXPOSE 8080

CMD ["./main"]
