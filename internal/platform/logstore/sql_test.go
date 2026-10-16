package logstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuote_EscapesEveryQuote(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "abc", `'abc'`},
		{"one quote", "it's", `'it''s'`},
		{"a closing attempt", "x' OR '1'='1", `'x'' OR ''1''=''1'`},
		{"a stacked statement", "x'; DROP TABLE nexul; --", `'x''; DROP TABLE nexul; --'`},
		{"only quotes", "''", `''''''`},
		{"a double quote and a percent are literal", `a"b%`, `'a"b%'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, quote(tt.in))
		})
	}
}

func TestSelectable_DropsBookkeepingSecretsAndUnsafeNames(t *testing.T) {
	schema := []string{"_timestamp", "body", "severity", "trace_id", "span_id", "severity_number", "flags", "dropped_attributes_count",
		"instrumentation_library_name", "api_token", "db_password", "client_secret", "ssh_key", "authorization", "cookie", "credentials",
		"environment", "Weird", `a"b`, "a b", "_all", "attrs_stack", "error"}
	assert.Equal(t, []string{"_timestamp", "body", "severity", "trace_id", "attrs_stack", "error"}, selectable(schema))
}

func TestSensitive(t *testing.T) {
	for _, name := range []string{"token", "API_TOKEN", "x_secret", "Password", "api_key", "auth", "set_cookie", "credential", "env", "environment"} {
		assert.True(t, sensitive(name), name)
	}
	for _, name := range []string{"body", "trace_id", "machine", "version", "user_id", "url", "error", "name"} {
		assert.False(t, sensitive(name), name)
	}
}

func TestWhere(t *testing.T) {
	all := func(string) bool { return true }
	only := func(cols ...string) func(string) bool {
		return func(c string) bool {
			for _, have := range cols {
				if c == have {
					return true
				}
			}
			return false
		}
	}
	tests := []struct {
		name     string
		q        Query
		have     func(string) bool
		want     string
		wantNone bool
	}{
		{"info drops debug", Query{Level: "info"}, all, " WHERE severity IN ('INFO', 'WARN', 'ERROR')", false},
		{"debug reads every level", Query{Level: "debug"}, all, "", false},
		{"warn", Query{Level: "warn"}, all, " WHERE severity IN ('WARN', 'ERROR')", false},
		{"error", Query{Level: "error"}, all, " WHERE severity IN ('ERROR')", false},
		{"a trace id is quoted", Query{Level: "debug", TraceID: "t'1"}, all, " WHERE trace_id = 't''1'", false},
		{"a name is quoted", Query{Level: "debug", Name: "edge"}, all, " WHERE name = 'edge'", false},
		{"text matches the message without wildcards", Query{Level: "debug", Text: "100%"}, all, " WHERE str_match_ignore_case(body, '100%')", false},
		{"server lines carry no source", Query{Level: "debug", Source: "server"}, all, " WHERE (source IS NULL OR source = 'server')", false},
		{"web", Query{Level: "debug", Source: "web"}, all, " WHERE source = 'web'", false},
		{"filters join with and", Query{Level: "error", Source: "runner", Name: "edge", TraceID: "t", Text: "x"}, all,
			" WHERE severity IN ('ERROR') AND trace_id = 't' AND name = 'edge' AND source = 'runner' AND str_match_ignore_case(body, 'x')", false},
		{"server needs no clause while no line has a source", Query{Level: "debug", Source: "server"}, only("body"), "", false},
		{"web matches nothing while no line has a source", Query{Level: "debug", Source: "web"}, only("body"), "", true},
		{"a trace id matches nothing before the column exists", Query{Level: "debug", TraceID: "t"}, only("body"), "", true},
		{"a name matches nothing before the column exists", Query{Level: "debug", Name: "n"}, only("body"), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, none := where(tt.q, tt.have)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantNone, none)
		})
	}
}

func TestSQL_NamesEveryColumnAndNeverStar(t *testing.T) {
	assert.Equal(t, `SELECT "_timestamp", "body" FROM "nexul" WHERE severity IN ('ERROR') ORDER BY _timestamp DESC`,
		listSQL([]string{"_timestamp", "body"}, " WHERE severity IN ('ERROR')"))
	assert.Equal(t, `SELECT count(*) AS c FROM "nexul"`, countSQL(""))
}
