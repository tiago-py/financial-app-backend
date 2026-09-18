.PHONY: dev down logs test fmt vet

dev:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f api

test:
	go test ./...

fmt:
	gofmt -w cmd internal

vet:
	go vet ./...
