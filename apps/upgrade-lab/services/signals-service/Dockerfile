FROM golang:1.22-alpine AS builder
WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY main.go main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o signals-service .

FROM alpine:3.20
RUN adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=builder /workspace/signals-service /app/signals-service
USER 10001
EXPOSE 8090
ENTRYPOINT ["/app/signals-service"]
