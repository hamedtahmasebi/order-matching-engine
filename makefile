migrate-up:
	migrate -source file://db/migrations -database postgres://postgres:postgres@localhost:5432/order_book?sslmode=disable up

migrate-down:
	migrate -source file://db/migrations -database postgres://postgres:postgres@localhost:5432/order_book?sslmode=disable down
