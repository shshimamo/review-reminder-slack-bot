build:
	go build -o bin/review-reminder ./cmd

run: build
	./bin/review-reminder

dev:
	go run ./cmd

docker-build:
	docker build -t review-reminder .

docker-run:
	docker run --env-file .env review-reminder
