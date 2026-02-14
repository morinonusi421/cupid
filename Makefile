.PHONY: help build test test-integration generate mocks deploy status logs restart reset-db-local reset-db migrate-up migrate-down migrate-status add-test-user db-copy db-cli db-open

help:
	@echo "Available commands:"
	@echo "  make build          - Build the Go binary locally"
	@echo "  make test           - Run tests (excluding entities/)"
	@echo "  make test-integration - Run backend integration tests (e2e/)"
	@echo "  make generate       - Generate entities from DB schema (sqlboiler)"
	@echo "  make mocks          - Generate mocks from interfaces (mockery)"
	@echo "  make migrate-up     - Run database migrations (local)"
	@echo "  make migrate-down   - Rollback last migration (local)"
	@echo "  make migrate-status - Show migration status (local)"
	@echo "  make deploy         - Deploy to EC2 (pull, build, restart)"
	@echo "  make status         - Check service status on EC2"
	@echo "  make logs           - Show service logs on EC2"
	@echo "  make restart        - Restart service on EC2"
	@echo "  make reset-db-local - Reset local database (cupid.db)"
	@echo "  make reset-db       - Reset database on EC2 (WARNING: destructive)"
	@echo "  make add-test-user  - Add test user to production DB (interactive)"
	@echo "  make db-copy        - Copy production DB to local (read-only)"
	@echo "  make db-cli         - Open SQLite CLI on EC2"
	@echo "  make db-open        - Copy DB and open with DB Browser for SQLite"

build:
	go build -o cupid ./cmd/server

test:
	go test $$(go list ./... | grep -v /entities)

test-integration:
	go test ./e2e -v

generate:
	sqlboiler sqlite3 --no-auto-timestamps

mocks:
	mockery

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

add-test-user:
	@echo "=== Add Test User to Production DB ==="
	@read -p "Name (カタカナ, 例: タナカタロウ): " name; \
	read -p "Birthday (YYYY-MM-DD, 例: 2000-01-15): " birthday; \
	read -p "LINE User ID (例: test123): " line_user_id; \
	read -p "Crush Name (カタカナ, 例: ヤマダハナコ): " crush_name; \
	read -p "Crush Birthday (YYYY-MM-DD, 例: 2000-05-20): " crush_birthday; \
	echo ""; \
	echo "Adding test user:"; \
	echo "  Name: $$name"; \
	echo "  Birthday: $$birthday"; \
	echo "  LINE User ID: $$line_user_id"; \
	echo "  Crush: $$crush_name ($$crush_birthday)"; \
	echo ""; \
	read -p "Proceed? (yes/no): " confirm; \
	if [ "$$confirm" != "yes" ]; then \
		echo "Aborted."; \
		exit 1; \
	fi; \
	ssh cupid-bot "sqlite3 ~/cupid/cupid.db \"INSERT INTO users (line_user_id, name, birthday, registration_step, crush_name, crush_birthday, matched_with_user_id, registered_at, updated_at) VALUES ('$$line_user_id', '$$name', '$$birthday', 2, '$$crush_name', '$$crush_birthday', NULL, datetime('now'), datetime('now'));\""; \
	echo "✅ Test user added successfully"

db-copy:
	@TIMESTAMP=$$(date +%Y%m%d_%H%M%S); \
	DB_FILE="$$HOME/Downloads/cupid_$$TIMESTAMP.db"; \
	echo "📥 Copying production DB from EC2..."; \
	scp cupid-bot:~/cupid/cupid.db "$$DB_FILE"; \
	echo "✅ DB copied to $$DB_FILE"

db-cli:
	@echo "🔌 Connecting to production DB (read-only recommended)..."
	@echo "💡 Tip: Use .tables, .schema, SELECT * FROM users;"
	ssh cupid-bot "cd ~/cupid && sqlite3 cupid.db"

db-open:
	@TIMESTAMP=$$(date +%Y%m%d_%H%M%S); \
	DB_FILE="$$HOME/Downloads/cupid_$$TIMESTAMP.db"; \
	echo "📥 Copying production DB from EC2..."; \
	scp cupid-bot:~/cupid/cupid.db "$$DB_FILE"; \
	echo "✅ DB copied to $$DB_FILE"; \
	if command -v db_browser_for_sqlite >/dev/null 2>&1 || [ -d "/Applications/DB Browser for SQLite.app" ]; then \
		echo "🚀 Opening with DB Browser for SQLite..."; \
		open -a "DB Browser for SQLite" "$$DB_FILE"; \
	else \
		echo "⚠️  DB Browser for SQLite not found."; \
		echo "💡 Install: brew install --cask db-browser-for-sqlite"; \
		echo "📂 Or open manually: $$DB_FILE"; \
	fi
