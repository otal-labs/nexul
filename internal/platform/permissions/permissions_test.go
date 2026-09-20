package permissions

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalog_GridShape(t *testing.T) {
	catalog := Catalog()
	require.Len(t, catalog, 69)
	assert.Equal(t, Info{Value: "docs:read", Label: "Read docs", Domain: "docs", Action: "read"}, catalog[0])
	assert.Equal(t, Info{Value: "docs:write", Label: "Create and update docs", Domain: "docs", Action: "write"}, catalog[1])
	assert.Equal(t, Info{Value: "docs:delete", Label: "Delete docs", Domain: "docs", Action: "delete"}, catalog[2])
	assert.Equal(t, Info{Value: "docs:thread", Label: "See doc threads", Domain: "docs", Action: "thread"}, catalog[3])
	assert.Equal(t, Action("audit:read"), catalog[len(catalog)-1].Value)

	seen := map[Action]bool{}
	domains := map[string]bool{}
	for _, info := range catalog {
		assert.False(t, seen[info.Value], "catalog repeats %q", info.Value)
		seen[info.Value] = true
		domains[info.Domain] = true
		assert.Equal(t, Action(info.Domain+":"+info.Action), info.Value)
	}
	assert.Len(t, domains, 28)
	assert.Equal(t, AllActions(), func() []Action {
		out := make([]Action, 0, len(catalog))
		for _, info := range catalog {
			out = append(out, info.Value)
		}
		return out
	}())
}

func TestCatalog_DomainDeclaredVerbs(t *testing.T) {
	catalog := Catalog()
	byValue := make(map[Action]Info, len(catalog))
	for _, info := range catalog {
		byValue[info.Value] = info
	}

	tests := []struct {
		value  Action
		label  string
		domain string
		action string
	}{
		{PlaysRead, "Read plays", "plays", "read"},
		{PlaysWrite, "Create and update plays", "plays", "write"},
		{PlaysDelete, "Delete plays", "plays", "delete"},
		{PlaysRun, "Run plays", "plays", "run"},
		{MemoriesRead, "Read memories", "memories", "read"},
		{MemoriesWrite, "Create and update memories", "memories", "write"},
		{MemoriesDelete, "Delete memories", "memories", "delete"},
		{MemoriesClone, "Clone memories to another project or workspace", "memories", "clone"},
		{DocsThread, "See doc threads", "docs", "thread"},
	}
	for _, tt := range tests {
		t.Run(string(tt.value), func(t *testing.T) {
			info, ok := byValue[tt.value]
			require.True(t, ok, "catalog is missing %q", tt.value)
			assert.Equal(t, Info{Value: tt.value, Label: tt.label, Domain: tt.domain, Action: tt.action}, info)
		})
	}
}

func TestParseAction(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Action
		ok   bool
	}{
		{"docs read", "docs:read", DocsRead, true},
		{"trimmed", "  roles:write ", RolesWrite, true},
		{"delete on a read-only domain", "events:delete", "", false},
		{"write on a read-only domain", "audit:write", "", false},
		{"legacy name", "manage_roles", "", false},
		{"legacy automation name", "automation.read", "", false},
		{"no row for auth", "auth:read", "", false},
		{"unknown", "comment", "", false},
		{"empty", "", "", false},
		{"plays run", "plays:run", PlaysRun, true},
		{"memories clone", "memories:clone", MemoriesClone, true},
		{"docs thread", "docs:thread", DocsThread, true},
		{"a verb another domain didn't declare", "plays:clone", "", false},
		{"a verb tickets didn't declare", "tickets:run", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseAction(tt.in)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestSet_HasWithWithout(t *testing.T) {
	t.Run("creator grant covers docs and sharing, nothing workspace-wide", func(t *testing.T) {
		for _, a := range []Action{DocsRead, DocsWrite, DocsDelete, PermissionsWrite} {
			assert.True(t, CreatorGrant.Has(a))
		}
		assert.False(t, CreatorGrant.Has(MembersWrite))
		assert.False(t, CreatorGrant.Has(RolesWrite))
		assert.False(t, CreatorGrant.Has("permissions:read"))
	})
	t.Run("empty set has none", func(t *testing.T) {
		for _, a := range AllActions() {
			assert.False(t, Set(nil).Has(a))
		}
	})
	t.Run("with adds, without removes, neither mutates the receiver", func(t *testing.T) {
		base := SetOf(DocsRead)
		s := base.With(DocsWrite)
		assert.True(t, s.Has(DocsRead))
		assert.True(t, s.Has(DocsWrite))
		assert.False(t, base.Has(DocsWrite))
		s2 := s.Without(DocsRead)
		assert.False(t, s2.Has(DocsRead))
		assert.True(t, s2.Has(DocsWrite))
		assert.True(t, s.Has(DocsRead))
	})
	t.Run("without on an absent action is a no-op", func(t *testing.T) {
		assert.Equal(t, SetOf(DocsRead), SetOf(DocsRead).Without(DocsDelete))
	})
}

func TestSetOf_NormalizesAndStrings(t *testing.T) {
	s := SetOf(DocsWrite, DocsRead, DocsWrite)
	assert.Equal(t, []Action{DocsRead, DocsWrite}, s.Actions())
	assert.Equal(t, "docs:read,docs:write", s.String())
	assert.Equal(t, "", SetOf().String())
}

func TestSet_JSON(t *testing.T) {
	tests := []struct {
		name string
		in   Set
		want string
	}{
		{"nil is an empty array", nil, `[]`},
		{"empty is an empty array", SetOf(), `[]`},
		{"values are a string array", SetOf(DocsWrite, DocsRead), `["docs:read","docs:write"]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.in)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
	t.Run("unmarshal normalizes", func(t *testing.T) {
		var s Set
		require.NoError(t, json.Unmarshal([]byte(`["docs:write","docs:read","docs:write"]`), &s))
		assert.Equal(t, SetOf(DocsRead, DocsWrite), s)
	})
	t.Run("unmarshal rejects non-arrays", func(t *testing.T) {
		var s Set
		require.Error(t, json.Unmarshal([]byte(`7`), &s))
	})
	t.Run("struct field round trip", func(t *testing.T) {
		type holder struct {
			Permissions Set `json:"permissions"`
		}
		b, err := json.Marshal(holder{})
		require.NoError(t, err)
		assert.Equal(t, `{"permissions":[]}`, string(b))
	})
}
