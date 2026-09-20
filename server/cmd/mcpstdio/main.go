// Command mcpstdio proxies stdin/stdout MCP JSON-RPC to a running server's HTTP MCP transport.
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/mcp"
)

func main() {
	connToken := os.Getenv("NEXUL_CONNECTION_TOKEN")
	pat := os.Getenv("NEXUL_PAT")
	secret := os.Getenv("NEXUL_AUTH_SECRET")
	override := os.Getenv("NEXUL_MCP_URL")

	if connToken == "" {
		fail("NEXUL_CONNECTION_TOKEN is required")
	}
	if pat == "" {
		fail("NEXUL_PAT is required")
	}
	if secret == "" {
		fail("NEXUL_AUTH_SECRET is required")
	}

	svc := auth.NewService(auth.Config{Secret: []byte(secret)})
	claims, err := svc.ParseConnectionToken(connToken)
	if err != nil {
		fail("invalid connection token: " + err.Error())
	}
	endpoint := override
	if endpoint == "" {
		endpoint = claims.MCPURL
	}
	if endpoint == "" {
		endpoint = claims.InstanceURL
	}
	if endpoint == "" {
		fail("connection token carries no server URL")
	}
	if !strings.HasSuffix(endpoint, "/mcp") {
		endpoint = strings.TrimSuffix(endpoint, "/") + "/mcp"
	}
	if err := validateEndpoint(endpoint); err != nil {
		fail(err.Error())
	}

	proxy := mcp.NewStdioProxy(endpoint, pat, nil)
	if err := proxy.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fail("proxy: " + err.Error())
	}
}

func validateEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("MCP endpoint must be an absolute http(s) URL, got %q", endpoint)
	}
	return nil
}

func fail(msg string) {
	_, _ = fmt.Fprintln(os.Stderr, "mcpstdio:", msg) // best-effort diagnostic; exit code carries the real result
	os.Exit(1)
}
