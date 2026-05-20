.PHONY: run build migrate-up migrate-down

run:
	go run ./cmd/main.go

build:
	go build -o bin/auth-service ./cmd/main.go

migrate-up:
	migrate -path ./migrations -database "postgres://postgres:secret@localhost:5432/auth_db?sslmode=disable" up

migrate-down:
	migrate -path ./migrations -database "postgres://postgres:secret@localhost:5432/auth_db?sslmode=disable" down