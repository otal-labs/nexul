package logstore

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShape_HeadlineFieldsAndAttrs(t *testing.T) {
	row := map[string]any{
		"_timestamp":  json.Number("1790000000123456"),
		"severity":    "ERROR",
		"body":        "deploy failed",
		"trace_id":    "t1",
		"source":      "runner",
		"name":        "edge",
		"machine":     "m1",
		"version":     "v1",
		"user_id":     "alice",
		"url":         "https://example.com/x",
		"error":       "exit 1",
		"count":       json.Number("3"),
		"missing":     nil,
		"api_token":   "hunter2",
		"db_password": "hunter2",
	}
	l := shape(row)
	assert.Equal(t, Line{
		Time: time.UnixMicro(1790000000123456).UTC(), Level: "error", Source: "runner", Message: "deploy failed", TraceID: "t1",
		Name: "edge", Machine: "m1", Version: "v1", UserID: "alice", URL: "https://example.com/x",
		Attrs: map[string]any{"error": "exit 1", "count": json.Number("3")},
	}, l)
}

func TestShape_ServerLinesHaveNoSource(t *testing.T) {
	assert.Equal(t, "server", shape(map[string]any{"body": "up"}).Source)
	assert.Equal(t, "server", shape(map[string]any{"body": "up", "source": ""}).Source)
	assert.Nil(t, shape(map[string]any{"body": "up"}).Attrs)
}

func TestShape_RedactsAndCuts(t *testing.T) {
	token := "ses_" + strings.Repeat("a", 50)
	l := shape(map[string]any{
		"body":  "failed with " + token,
		"url":   "wss://example.com/live?token=abc123",
		"error": "bearer " + token,
		"long":  strings.Repeat("é", maxChars+10),
	})
	assert.Equal(t, "failed with [redacted token]", l.Message)
	assert.Equal(t, "wss://example.com/live?token=[redacted token]", l.URL)
	assert.Equal(t, "bearer [redacted token]", l.Attrs["error"])
	long := l.Attrs["long"].(string)
	assert.Equal(t, maxChars+1, len([]rune(long)), "cut at 2000 characters, with an ellipsis")
	assert.True(t, strings.HasSuffix(long, "…"))
}

func TestShape_MessageIsCut(t *testing.T) {
	l := shape(map[string]any{"body": strings.Repeat("x", maxChars+1)})
	require.Len(t, []rune(l.Message), maxChars+1)
}

// A relayed runner or host line is written with these columns (logging.Relay), and the reader files each one.
func TestShape_ARelayedLineKeepsItsStampedColumns(t *testing.T) {
	l := shape(map[string]any{
		"_timestamp": json.Number("1790000000123456"), "severity": "WARN", "body": "pull failed",
		"source": "runner", "name": "edge", "source_id": "r-1", "machine": "box", "version": "v1", "image": "acme/api",
	})

	assert.Equal(t, Line{
		Time: time.UnixMicro(1790000000123456).UTC(), Level: "warn", Source: "runner", Message: "pull failed",
		Name: "edge", Machine: "box", Version: "v1", Attrs: map[string]any{"source_id": "r-1", "image": "acme/api"},
	}, l)
}
