package deploy

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestSlug(t *testing.T) {
	cases := []struct {
		name   string
		branch string
		maxLen int
		want   string
	}{
		{"lowercases", "Feature/Discord", 0, "feature-discord"},
		{"collapses runs of non-alnum", "feature//discord--bot", 0, "feature-discord-bot"},
		{"trims leading and trailing dashes", "-feature-", 0, "feature"},
		{"caps length", strings.Repeat("a", 100), 10, "aaaaaaaaaa"},
		{"trims a dash exposed by capping", "feature-" + strings.Repeat("a", 10), 8, "feature"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Slug(tc.branch, tc.maxLen))
		})
	}
}

func TestBranchDeployRule_Validate(t *testing.T) {
	t.Run("empty pattern is invalid", func(t *testing.T) {
		err := BranchDeployRule{DockerNetwork: "app-net"}.Validate()
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("multiple wildcards are invalid", func(t *testing.T) {
		err := BranchDeployRule{Pattern: "feature/*/*", DockerNetwork: "app-net"}.Validate()
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("a leading wildcard is invalid", func(t *testing.T) {
		err := BranchDeployRule{Pattern: "*/feature", DockerNetwork: "app-net"}.Validate()
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing docker network is invalid", func(t *testing.T) {
		err := BranchDeployRule{Pattern: "main"}.Validate()
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("a wildcard rule with an explicit name suffix is invalid", func(t *testing.T) {
		err := BranchDeployRule{Pattern: "feature/*", DockerNetwork: "app-net", NameSuffix: "qa"}.Validate()
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("an invalid name suffix is rejected", func(t *testing.T) {
		err := BranchDeployRule{Pattern: "dev", DockerNetwork: "app-net", NameSuffix: "QA env"}.Validate()
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("a hostname template with no port is invalid", func(t *testing.T) {
		err := BranchDeployRule{Pattern: "main", DockerNetwork: "app-net", HostnameTemplate: "{branch}.example.com"}.Validate()
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("an exact rule with no name suffix is valid (in place)", func(t *testing.T) {
		assert.NoError(t, BranchDeployRule{Pattern: "main", DockerNetwork: "app-net"}.Validate())
	})
	t.Run("a wildcard rule is valid", func(t *testing.T) {
		assert.NoError(t, BranchDeployRule{Pattern: "feature/*", DockerNetwork: "app-net"}.Validate())
	})
	t.Run("a hostname template with a port is valid", func(t *testing.T) {
		assert.NoError(t, BranchDeployRule{Pattern: "main", DockerNetwork: "app-net", HostnameTemplate: "{branch}.example.com", Port: 8080}.Validate())
	})
	t.Run("overrides on an in-place rule are invalid", func(t *testing.T) {
		err := BranchDeployRule{Pattern: "main", DockerNetwork: "app-net", Overrides: map[string]string{"DATABASE_URL": "x"}}.Validate()
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("an override with an empty key is invalid", func(t *testing.T) {
		err := BranchDeployRule{Pattern: "feature/*", DockerNetwork: "app-net", Overrides: map[string]string{" ": "x"}}.Validate()
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("overrides on a cloning rule are valid", func(t *testing.T) {
		assert.NoError(t, BranchDeployRule{Pattern: "dev", DockerNetwork: "qa-net", NameSuffix: "qa", Overrides: map[string]string{"DATABASE_URL": "x"}}.Validate())
		assert.NoError(t, BranchDeployRule{Pattern: "feature/*", DockerNetwork: "qa-net", Overrides: map[string]string{"DATABASE_URL": "x"}}.Validate())
	})
}

func TestBranchDeployRule_ApplyOverrides(t *testing.T) {
	base := map[string]string{"PORT": "8080", "DATABASE_URL": "postgres://prod"}
	tests := []struct {
		name      string
		overrides map[string]string
		want      map[string]string
	}{
		{"no overrides keeps the base values", nil, map[string]string{"PORT": "8080", "DATABASE_URL": "postgres://prod"}},
		{"an override replaces the base value", map[string]string{"DATABASE_URL": "postgres://qa"}, map[string]string{"PORT": "8080", "DATABASE_URL": "postgres://qa"}},
		{"an override adds a key the base lacks", map[string]string{"DEBUG": "1"}, map[string]string{"PORT": "8080", "DATABASE_URL": "postgres://prod", "DEBUG": "1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BranchDeployRule{Pattern: "feature/*", Overrides: tt.overrides}.ApplyOverrides(base)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, "postgres://prod", base["DATABASE_URL"], "the base map is never modified")
		})
	}
}

func TestBranchDeployRule_Matches(t *testing.T) {
	t.Run("exact pattern matches only that branch", func(t *testing.T) {
		r := BranchDeployRule{Pattern: "main"}
		assert.True(t, r.Matches("main"))
		assert.False(t, r.Matches("mainline"))
	})
	t.Run("wildcard matches the prefix", func(t *testing.T) {
		r := BranchDeployRule{Pattern: "feature/*"}
		assert.True(t, r.Matches("feature/discord-integration"))
		assert.False(t, r.Matches("main"))
		assert.False(t, r.Matches("feature")) // no trailing slash, no match
	})
}

func TestBranchDeployRule_DerivesCloneAndCloneSuffix(t *testing.T) {
	t.Run("exact rule with empty suffix deploys in place", func(t *testing.T) {
		r := BranchDeployRule{Pattern: "main", DockerNetwork: "app-net"}
		assert.False(t, r.DerivesClone())
	})
	t.Run("exact rule with a suffix derives one stable clone", func(t *testing.T) {
		r := BranchDeployRule{Pattern: "dev", DockerNetwork: "app-net", NameSuffix: "qa"}
		assert.True(t, r.DerivesClone())
		assert.Equal(t, "qa", r.CloneSuffix("dev"))
	})
	t.Run("wildcard rule always derives, keyed by the branch slug", func(t *testing.T) {
		r := BranchDeployRule{Pattern: "feature/*", DockerNetwork: "app-net"}
		assert.True(t, r.DerivesClone())
		assert.Equal(t, "feature-discord-integration", r.CloneSuffix("feature/discord-integration"))
	})
}

func TestStack_Validate_BranchDeployRules(t *testing.T) {
	t.Run("wildcard rule requires a build source", func(t *testing.T) {
		stack := validStack()
		stack.BranchDeployRules = []BranchDeployRule{{Pattern: "feature/*", DockerNetwork: "app-net"}}
		err := stack.Validate()
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("wildcard rule is fine with a build source", func(t *testing.T) {
		stack := validStack()
		stack.BuildSource = &BuildSource{RepoOwner: "acme", RepoName: "api"}
		stack.BranchDeployRules = []BranchDeployRule{{Pattern: "feature/*", DockerNetwork: "app-net"}}
		assert.NoError(t, stack.Validate())
	})
	t.Run("an invalid rule fails validation", func(t *testing.T) {
		stack := validStack()
		stack.BranchDeployRules = []BranchDeployRule{{Pattern: "", DockerNetwork: "app-net"}}
		err := stack.Validate()
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}
