# ShopAPI

A REST API for a home-appliance store, built with Go, chi, and PostgreSQL (pgx/v5).

Implements clean layered architecture (handler → service → repository) with DTO/entity
separation, transactional operations, input validation, and a normalized relational schema.
Built as a learning project to practice idiomatic Go backend development: dependency
inversion via interfaces, database transactions, and RESTful API design.

## Features
- Full CRUD for clients, products, suppliers, and product images
- PostgreSQL schema normalized to 3NF, managed via golang-migrate
- Transactional writes (e.g. client creation with address, address updates)
- Request validation with detailed 400 error responses
- Clean separation of concerns: entity (DAL) / dto (API contracts) / service (business logic) / repository (data access)
- Dependency injection via interfaces defined by consumers (idiomatic Go style)

## Tech stack
Go · chi · pgx/v5 (pgxpool) · PostgreSQL · golang-migrate · Docker Compose · go-playground/validator

## Tests
Written with [testify](https://github.com/stretchr/testify) (`assert`, `require`, `mock`):
- unit tests for handlers (`httptest`), middleware, mappers and services, with mocked dependencies;
- gRPC auth client tests against an in-memory server (`bufconn`);
- integration tests for the PostgreSQL repositories using [testcontainers-go](https://golang.testcontainers.org/) (requires Docker).

```bash
go test -race -cover ./...
```
