.PHONY: dev dev-frontend build test test-race test-frontend lint vet vuln check clean docker docker-nas ci

dev:
	go run ./cmd/server

dev-frontend:
	cd web && npm run dev

build:
	go build -o build/md ./cmd/server

test:
	go test ./...

test-race:
	go test -race -timeout 120s ./...

test-frontend:
	cd web && npm test

lint:
	golangci-lint run

vet:
	go vet ./...

vuln:
	govulncheck ./...

check:
	cd web && npm run check

clean:
	rm -rf build/ web/dist/ coverage/

docker:
	docker compose up --build

docker-nas:
	docker compose -f docker-compose.nas.yml up -d --build

ci: vet lint test check test-frontend
