package tickets

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func callTool(t *testing.T, ctx context.Context, tools []mcptool.Tool, name, args string) (any, error) {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

func TestMCPTools_Names(t *testing.T) {
	var names []string
	for _, tool := range MCPTools(newTestService(newFakeRepo())) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
	}
	assert.Equal(t, []string{"ticket_delete", "ticket_test_report"}, names)
}

func TestTicketDeleteTool(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr error
	}{
		{"missing id is invalid", `{}`, apperrs.ErrInvalid},
		{"an unknown argument is invalid", `{"id":"t1","force":true}`, apperrs.ErrInvalid},
		{"a missing ticket is not found", `{"id":"nope"}`, apperrs.ErrNotFound},
		{"a missing key is not found", `{"id":"REF-9"}`, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTestingFixture("qa")
			_, err := callTool(t, t.Context(), MCPTools(f.svc), "ticket_delete", tt.args)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
	t.Run("deletes by key and returns what it deleted", func(t *testing.T) {
		f := newTestingFixture("qa")
		f.repo.prefixes = map[string]string{"p1": "REF"}
		f.repo.tickets["t1"].Number = 102
		got, err := callTool(t, t.Context(), MCPTools(f.svc), "ticket_delete", `{"id":"REF-102"}`)
		require.NoError(t, err)
		assert.Equal(t, mcptool.Gone("t1"), got)
		assert.NotContains(t, f.repo.tickets, "t1")
	})
}

func TestTicketTestReportTool_Errors(t *testing.T) {
	tests := []struct {
		name    string
		ctx     func(context.Context) context.Context
		args    string
		wantErr error
	}{
		{"missing outcome is invalid", asUser, `{"id":"t1"}`, apperrs.ErrInvalid},
		{"an unknown outcome is invalid", asUser, `{"id":"t1","outcome":"flaky"}`, apperrs.ErrInvalid},
		{"a screenshot that is not a string is invalid", asUser, `{"id":"t1","outcome":"fail","actual":"x","screenshots":[1]}`, apperrs.ErrInvalid},
		{"a fail without actual is invalid", asUser, `{"id":"t1","outcome":"fail"}`, apperrs.ErrInvalid},
		{"a missing ticket is not found", asUser, `{"id":"nope","outcome":"pass"}`, apperrs.ErrNotFound},
		{"no signed-in tester is unauthorized", func(ctx context.Context) context.Context { return ctx }, `{"id":"t1","outcome":"pass"}`, apperrs.ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTestingFixture("qa")
			_, err := callTool(t, tt.ctx(t.Context()), MCPTools(f.svc), "ticket_test_report", tt.args)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, f.threads.posts)
		})
	}
}

func TestTicketTestReportTool(t *testing.T) {
	t.Run("fail posts the report and moves the ticket back to progress", func(t *testing.T) {
		f := newTestingFixture("qa")
		got, err := callTool(t, asUser(t.Context()), MCPTools(f.svc), "ticket_test_report",
			`{"id":"t1","outcome":"fail","steps":"s","expected":"e","actual":"broken","screenshots":["att-9"]}`)
		require.NoError(t, err)
		assert.Equal(t, testReportResult{ID: "t1", Outcome: "fail", StatusID: "build"}, got)
		require.Len(t, f.threads.posts, 1)
		assert.True(t, strings.HasPrefix(f.threads.posts[0].body, "Test failed by Nexul · for onik97\n"))
		assert.Contains(t, f.threads.posts[0].body, "![screenshot](/api/attachments/att-9)")
	})
	t.Run("pass moves the ticket to done and makes the caller its tester", func(t *testing.T) {
		f := newTestingFixture("qa")
		got, err := callTool(t, asUser(t.Context()), MCPTools(f.svc), "ticket_test_report", `{"id":"t1","outcome":"pass"}`)
		require.NoError(t, err)
		assert.Equal(t, testReportResult{ID: "t1", Outcome: "pass", StatusID: "shipped", Tester: "onik97"}, got)
		assert.Equal(t, []threadPost{{ticketID: "t1", authorID: "u-1", body: "Passed by Nexul · for onik97"}}, f.threads.posts)
	})
}
