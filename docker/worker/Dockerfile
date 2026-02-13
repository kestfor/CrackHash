FROM golang:1.25-alpine AS builder

#RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o worker ./cmd/worker

FROM alpine:latest

#RUN apk --no-cache add ca-certificates

RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser

WORKDIR /app

COPY --from=builder /build/worker .
COPY --from=builder /build/configs/worker.yaml .

RUN chown -R appuser:appgroup /app

USER appuser

CMD ["./worker", "--config=worker.yaml"]