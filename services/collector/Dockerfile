FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY *.go ./
RUN go build -o collector .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/collector .
EXPOSE 8081
CMD ["./collector"]
