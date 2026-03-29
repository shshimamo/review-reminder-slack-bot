FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o review-reminder ./cmd

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -g '' appuser
USER appuser
COPY --from=builder /app/review-reminder /app/review-reminder
ENTRYPOINT ["/app/review-reminder"]
