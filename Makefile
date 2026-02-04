.PHONY: help build test test-integration generate deploy status logs restart reset-db-local reset-db migrate-up migrate-down migrate-status

help:
	@echo "Available commands:"
	@echo "  make build          - Build the Go binary locally"
	@echo "  make test           - Run tests (excluding entities/)"
	@echo "  make test-integration - Run backend integration tests (e2e/)"
	@echo "  make generate       - Generate entities from DB schema (sqlboiler)"
	@echo "  make migrate-up     - Run database migrations (local)"
	@echo "  make migrate-down   - Rollback last migration (local)"
	@echo "  make migrate-status - Show migration status (local)"
	@echo "  make deploy         - Deploy to EC2 (pull, build, restart)"
	@echo "  make status         - Check service status on EC2"
	@echo "  make logs           - Show service logs on EC2"
	@echo "  make restart        - Restart service on EC2"
	@echo "  make reset-db-local - Reset local database (cupid.db)"
	@echo "  make reset-db       - Reset database on EC2 (WARNING: destructive)"

build:
	go build -o cupid ./cmd/server

test:
	go test $$(go list ./... | grep -v /entities)

test-integration:
	go test ./e2e -v

generate:
	sqlboiler sqlite3 --no-auto-timestamps

deploy:
	ssh cupid-bot "bash -l -c 'cd ~/cupid && git pull && sql-migrate up -config=db/dbconfig.yml && go build -o cupid ./cmd/server && sudo systemctl restart cupid && sudo systemctl status cupid'"

status:
	ssh cupid-bot "sudo systemctl status cupid"

logs:
	ssh cupid-bot "sudo journalctl -u cupid -n 50 --no-pager"

restart:
	ssh cupid-bot "sudo systemctl restart cupid && sudo systemctl status cupid"

reset-db-local:
	@echo "⚠️  WARNING: This will delete cupid.db and reset all data!"
	@read -p "Are you sure? (yes/no): " confirm && [ "$$confirm" = "yes" ] || (echo "Aborted." && exit 1)
	@echo "Removing cupid.db..."
	rm -f cupid.db
	@echo "✅ Database reset. Run 'make build && ./cupid' to recreate tables."

reset-db:
	@echo "⚠️  WARNING: This will delete cupid.db on EC2 and reset all data!"
	@read -p "Are you sure? (yes/no): " confirm && [ "$$confirm" = "yes" ] || (echo "Aborted." && exit 1)
	@echo "Stopping service, removing DB, running migrations, restarting service..."
	ssh cupid-bot "bash -l -c 'cd ~/cupid && sudo systemctl stop cupid && rm -f cupid.db && sql-migrate up -config=db/dbconfig.yml && sudo systemctl start cupid && sleep 2 && sudo systemctl status cupid'"
	@echo "Checking database..."
	ssh cupid-bot "bash -l -c 'cd ~/cupid && sqlite3 cupid.db .tables'"

migrate-up:
	sql-migrate up -config=db/dbconfig.yml

migrate-down:
	sql-migrate down -config=db/dbconfig.yml

migrate-status:
	sql-migrate status -config=db/dbconfig.yml
