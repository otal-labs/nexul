# Nexul build targets (ws-11).
# Binaries are static (CGO_ENABLED=0) — server and runner have no cgo deps
# (modernc.org/sqlite is pure Go, per ws-01).

BIN_DIR := dist
VERSION ?= dev
GO_LDFLAGS := -ldflags="-s -w -X github.com/otal-labs/nexul/internal/platform/version.Version=$(VERSION)"
# Version is pinned in go.mod's tool directive; local and CI both resolve it from there.
SQLC ?= go tool sqlc

.PHONY: build build-server build-runner build-web build-single test vet lint vuln coverage sqlc sqlc-check clean

build: build-server build-runner build-web

build-server:
	CGO_ENABLED=0 go build $(GO_LDFLAGS) -o $(BIN_DIR)/nexul-server ./server/cmd

build-runner:
	CGO_ENABLED=0 go build $(GO_LDFLAGS) -o $(BIN_DIR)/nexul-runner ./runner/cmd

build-web:
	bun run --cwd web build

# Single-binary server: embeds web/dist via go:embed (server/webui). The SPA
# is served by the binary itself; no nginx, no node at runtime.
build-single: build-web
	@rm -rf server/webui/dist
	@mkdir -p server/webui/dist
	@cp -r web/dist/. server/webui/dist/
	CGO_ENABLED=0 go build -tags embed $(GO_LDFLAGS) -o $(BIN_DIR)/nexul-server ./server/cmd

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

vuln:
	go tool govulncheck ./...

coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	LC_ALL=C awk 'BEGIN { print "mode: atomic" } /^mode:/ { next } $$1 ~ /\/cmd\/|\/testutil\/|\/sqlcgen\// { next } { print }' coverage.out > coverage.filtered.out
	LC_ALL=C awk '/^mode:/ { next } { total += $$2; if ($$3 > 0) covered += $$2 } END { c = total ? covered * 100 / total : 100; printf "Coverage (exempt paths excluded): %.1f%% (threshold 80%%)\n", c; exit !(c >= 80) }' coverage.filtered.out
	go tool cover -html=coverage.filtered.out -o coverage.html

# Regenerates internal/platform/storage/sqlcgen from queries/*.sql against the migrations.
sqlc:
	$(SQLC) generate

# Fails when the committed generated code is stale or a query no longer matches the schema.
sqlc-check:
	$(SQLC) vet
	$(SQLC) diff

clean:
	rm -rf $(BIN_DIR) server/webui/dist
