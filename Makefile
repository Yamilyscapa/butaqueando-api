APP_NAME := butaqueando-api

.PHONY: run test tidy db-schema db-seed db-seed-media db-bootstrap backfill-media

run:
	go run ./cmd/api

test:
	go test ./...

tidy:
	go mod tidy

db-schema:
	bash -c 'if [ -f .env ]; then set -a && source .env && set +a; fi; if [ -z "$$DATABASE_URL" ]; then exit 1; fi; psql "$$DATABASE_URL" -f db/schema.sql'

db-seed:
	bash -c 'if [ -f .env ]; then set -a && source .env && set +a; fi; if [ -z "$$DATABASE_URL" ]; then exit 1; fi; psql "$$DATABASE_URL" -f db/seeds/seed.sql'

db-seed-media:
	bash -c 'if [ -f .env ]; then set -a && source .env && set +a; fi; go run ./cmd/seedmedia'

db-bootstrap: db-schema db-seed db-seed-media

backfill-media:
	bash -c 'if [ -f .env ]; then set -a && source .env && set +a; fi; go run ./cmd/backfillmedia'
