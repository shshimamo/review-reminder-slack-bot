FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o review-reminder-slack-bot ./cmd

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -g '' appuser
USER appuser
ENTRYPOINT ["/app/review-reminder-slack-bot"]
COPY --from=builder /app/review-reminder-slack-bot /app/review-reminder-slack-bot
