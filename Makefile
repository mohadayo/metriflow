.PHONY: test test-go test-python test-ts lint up down build clean

test: test-go test-python test-ts

test-go:
	cd services/collector && go test -v ./...

test-python:
	cd services/analyzer && pip install -q -r requirements.txt && pytest -v

test-ts:
	cd services/gateway && npm install && npm test

lint: lint-go lint-python lint-ts

lint-go:
	cd services/collector && go vet ./...

lint-python:
	cd services/analyzer && pip install -q -r requirements.txt && flake8 --max-line-length=100 app.py test_app.py

lint-ts:
	cd services/gateway && npm install && npx eslint src/

build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

clean:
	docker compose down -v --rmi local
	rm -rf services/gateway/node_modules services/gateway/dist
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
