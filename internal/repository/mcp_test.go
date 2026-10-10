package repository

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/jsonx"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func callTool(t *testing.T, s *fakeScanner, name, args string) (any, error) {
	t.Helper()
	for _, tool := range MCPTools(newTestService(s, nil)) {
		if tool.Name == name {
			return tool.Call(t.Context(), json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

func TestMCPTools_Surface(t *testing.T) {
	var names []string
	for _, tool := range MCPTools(newTestService(&fakeScanner{}, nil)) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
		assert.NotEmpty(t, tool.Description, tool.Name)
	}
	assert.Equal(t, []string{"repository_list", "repository_scan"}, names)
}

func TestMCPTools_Errors(t *testing.T) {
	tests := []struct {
		name, tool, args string
		scanner          *fakeScanner
		want             error
		msg              string
	}{
		{"repository_list surfaces a provider failure", "repository_list", `{}`, &fakeScanner{listErr: apperrors.ErrUnauthorized}, apperrors.ErrUnauthorized, ""},
		{"repository_list surfaces an installations failure", "repository_list", `{"installations":true}`, &fakeScanner{installErr: apperrors.ErrUnauthorized}, apperrors.ErrUnauthorized, ""},
		{"repository_scan needs an owner", "repository_scan", `{"repo":"api"}`, &fakeScanner{}, apperrors.ErrInvalid, ""},
		{"repository_scan takes repo, not name", "repository_scan", `{"owner":"acme","name":"api"}`, &fakeScanner{}, apperrors.ErrInvalid, ""},
		{"repository_scan of a missing repository", "repository_scan", `{"owner":"acme","repo":"ghost"}`, &fakeScanner{treeErr: apperrors.ErrNotFound}, apperrors.ErrNotFound, "repository_list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(t, tt.scanner, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.want)
			assert.Contains(t, err.Error(), tt.msg)
		})
	}
}

func TestRepositoryList_Pages(t *testing.T) {
	s := &fakeScanner{repos: []Repo{{ID: 1, FullName: "acme/api"}, {ID: 2, FullName: "acme/web"}}}
	got, err := callTool(t, s, "repository_list", `{"limit":1}`)
	require.NoError(t, err)
	page := got.(mcptool.Page[Repo])
	require.Len(t, page.Items, 1)
	assert.Equal(t, "acme/api", page.Items[0].FullName)
	assert.Equal(t, 2, page.Total)
}

func TestRepositoryList_SearchesByQ(t *testing.T) {
	s := &fakeScanner{repos: []Repo{{ID: 1, FullName: "acme/api"}, {ID: 2, FullName: "acme/web"}}}
	got, err := callTool(t, s, "repository_list", `{"q":"WEB","refresh":true}`)
	require.NoError(t, err)
	page := got.(mcptool.Page[Repo])
	require.Len(t, page.Items, 1)
	assert.Equal(t, "acme/web", page.Items[0].FullName)
	assert.True(t, s.lastRefresh)
}

func TestRepositoryList_Installations(t *testing.T) {
	s := &fakeScanner{
		repos:    []Repo{{ID: 1, FullName: "acme/api", Provider: "github"}},
		installs: []Installation{{ID: 9, AccountLogin: "acme", AccountType: "organization", RepositorySelection: "all"}},
	}
	got, err := callTool(t, s, "repository_list", `{"installations":true}`)
	require.NoError(t, err)
	b, err := jsonx.Marshal(got)
	require.NoError(t, err)
	assert.JSONEq(t, `{"items":[{"id":1,"owner":"","name":"","full_name":"acme/api","default_branch":"","html_url":"","provider":"github"}],"total":1,"has_more":false,
		"installations":[{"id":9,"account_login":"acme","account_type":"organization","account_avatar_url":"","repository_selection":"all","html_url":"","workspaces":[]}]}`, string(b))
}

func TestRepositoryScan(t *testing.T) {
	s := &fakeScanner{
		tree:  []TreeEntry{{Path: "docker-compose.yml", Type: "blob"}},
		files: map[string][]byte{"docker-compose.yml": []byte("services:\n  web:\n    image: nginx\n    ports:\n      - \"80:80\"\n")},
	}
	got, err := callTool(t, s, "repository_scan", `{"owner":"acme","repo":"api","ref":"dev"}`)
	require.NoError(t, err)
	result := got.(*ScanResult)
	assert.Equal(t, "dev", result.DefaultBranch)
	require.Len(t, result.Candidates, 1)
	assert.Equal(t, KindCompose, result.Candidates[0].Kind)
}
