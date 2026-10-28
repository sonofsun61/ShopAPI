include .env
export

BIN_DIR := $(CURDIR)/bin
MIGRATE_VERSION ?= v4.20.1
MIGRATE := $(BIN_DIR)/migrate-$(MIGRATE_VERSION)

export

.PHONY: db-up db-down migrate-up migrate-down lint test wire run swag

run: db-up migrate-up
	@go run ./cmd/ShopAPI/main.go

db-up:
	@docker compose up -d

db-down:
	@docker compose down

$(MIGRATE):
	@GOBIN=$(BIN_DIR) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)
	@mv $(BIN_DIR)/migrate $(MIGRATE)

migrate-up: $(MIGRATE)
	@$(MIGRATE) -path migrations -database "$$DATABASE_URL" up

migrate-down: $(MIGRATE)
	@$(MIGRATE) -path migrations -database "$$DATABASE_URL" down 1

lint:
	@go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "не отформатированы"; gofmt -l .; exit 1)

test:
	@go test -race -cover -count=1 ./...

wire:
	@cd internal/app && go tool wire

swag:
	@go tool swag init -g cmd/ShopAPI/main.go