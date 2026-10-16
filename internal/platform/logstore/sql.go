package logstore

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// stream is the OpenObserve stream the server's OTLP exporter writes to (the stream-name header).
const stream = "nexul"

// columnName is what OpenObserve normalises every key to, so a schema column that is not one is never selected.
var columnName = regexp.MustCompile(`^[a-z0-9_]+$`)

// bookkeeping are the columns OpenObserve and the OTLP bridge add that tell a reader nothing.
var bookkeeping = []string{"span_id", "severity_number", "flags", "dropped_attributes_count", "instrumentation_library_name"}

// selectable is the schema's columns a line may carry: everything but bookkeeping and anything named like a secret.
func selectable(schema []string) []string {
	var cols []string
	for _, c := range schema {
		if !columnName.MatchString(c) || slices.Contains(bookkeeping, c) || sensitive(c) {
			continue
		}
		if strings.HasPrefix(c, "_") && c != "_timestamp" {
			continue
		}
		cols = append(cols, c)
	}
	return cols
}

// sensitive reports whether an attribute's name says its value is a secret.
func sensitive(name string) bool {
	name = strings.ToLower(name)
	for _, part := range []string{"token", "secret", "password", "key", "auth", "cookie", "credential", "env"} {
		if strings.Contains(name, part) {
			return true
		}
	}
	return false
}

// quote writes v as a SQL string; the API has no bind parameters. Query checks values for backslashes first.
func quote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

// where is the filter q asks for. none is true when a filter names a column the stream does not have yet, so no
// line can match and no search is needed.
func where(q Query, have func(string) bool) (clause string, none bool) {
	var conds []string
	if i := slices.Index(levels, q.Level); i > 0 {
		var in []string
		for _, l := range levels[i:] {
			in = append(in, quote(strings.ToUpper(l)))
		}
		conds = append(conds, "severity IN ("+strings.Join(in, ", ")+")")
	}
	for _, f := range []struct{ column, value string }{{"trace_id", q.TraceID}, {"name", q.Name}} {
		if f.value == "" {
			continue
		}
		if !have(f.column) {
			return "", true
		}
		conds = append(conds, f.column+" = "+quote(f.value))
	}
	switch {
	case q.Source == "":
	case !have("source"):
		if q.Source != "server" {
			return "", true
		}
	case q.Source == "server":
		conds = append(conds, "(source IS NULL OR source = 'server')")
	default:
		conds = append(conds, "source = "+quote(q.Source))
	}
	if q.Text != "" {
		conds = append(conds, "str_match_ignore_case(body, "+quote(q.Text)+")")
	}
	if len(conds) == 0 {
		return "", false
	}
	return " WHERE " + strings.Join(conds, " AND "), false
}

// listSQL reads the page's lines, newest first.
func listSQL(cols []string, clause string) string {
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = `"` + c + `"`
	}
	return fmt.Sprintf(`SELECT %s FROM "%s"%s ORDER BY _timestamp DESC`, strings.Join(quoted, ", "), stream, clause)
}

func countSQL(clause string) string {
	return fmt.Sprintf(`SELECT count(*) AS c FROM "%s"%s`, stream, clause)
}
