# Nexul build targets (ws-11).
# Binaries are static (CGO_ENABLED=0) — server and runner have no cgo deps
# (modernc.org/sqlite is pure Go, per ws-01).

BIN_DIR := dist
VERSION ?= dev
GO_LDFLAGS := -ldflags="-s -w -X github.com/otal-labs/nexul/internal/platform/version.Version=$(VERSION)"
# Version is pinned in go.mod's tool directive; local and CI both resolve it from there.
SQLC ?= go tool sqlc

.PHONY: build build-cli build-server build-runner build-web build-single test vet lint vuln coverage sqlc sqlc-check live-topics clean

build: build-cli build-server build-runner build-web

build-cli:
	CGO_ENABLED=0 go build $(GO_LDFLAGS) -o $(BIN_DIR)/nexul ./cmd/nexul

build-server:
	CGO_ENABLED=0 go build $(GO_LDFLAGS) -o $(BIN_DIR)/nexul-server ./server/cmd

build-runner:
	CGO_ENABLED=0 go build $(GO_LDFLAGS) -o $(BIN_DIR)/nexul-runner ./runner/cmd

build-web:
	bun run --cwd web build

# The nexul-server binary as released: embeds web/dist via go:embed (server/webui). The SPA
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

# The storage package skips -race: it instruments the pure-Go SQLite engine (24s becomes ~10min); server/cmd races storage.
coverage:
	go test -race -coverprofile=coverage.race.out -covermode=atomic $$(go list ./... | sed '/\/internal\/platform\/storage$$/d')
	go test -coverprofile=coverage.storage.out -covermode=atomic ./internal/platform/storage/
	cat coverage.race.out coverage.storage.out > coverage.out
	LC_ALL=C awk 'BEGIN { print "mode: atomic" } /^mode:/ { next } $$1 ~ /\/cmd\/|\/testutil\/|\/sqlcgen\/|\/t3rpctest\// { next } { print }' coverage.out > coverage.filtered.out
	LC_ALL=C awk '/^mode:/ { next } { total += $$2; if ($$3 > 0) covered += $$2 } END { c = total ? covered * 100 / total : 100; printf "Coverage (exempt paths excluded): %.1f%% (threshold 80%%)\n", c; exit !(c >= 80) }' coverage.filtered.out
	go tool cover -html=coverage.filtered.out -o coverage.html

# Regenerates internal/platform/storage/sqlcgen from internal/platform/storage/queries/*.sql against the migrations.
sqlc:
	$(SQLC) generate

# Fails when the committed generated code is stale or a query no longer matches the schema.
sqlc-check:
	$(SQLC) vet
	$(SQLC) diff

# Rewrites the browser's copy of the topics the live socket pushes, from the server's audience rules.
live-topics:
	go test ./server/cmd -run TestLiveTopicsFile_MatchesTheRules -update-live-topics

clean:
	rm -rf $(BIN_DIR) server/webui/dist
