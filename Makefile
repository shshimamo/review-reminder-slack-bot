build:
	go build -o bin/review-reminder ./cmd

run: build
	./bin/review-reminder

dev:
	go run ./cmd
