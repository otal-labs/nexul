package deploy

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTargetStack(rules ...BranchDeployRule) *Stack {
	return &Stack{
		ID: "s1", ProjectID: "p1", Name: "web", Slug: "web", Strategy: StrategyRun, DockerNetwork: "prod_net",
		BuildSource:       &BuildSource{RepoOwner: "acme", RepoName: "app", Branch: "main", Dockerfile: "Dockerfile"},
		BranchDeployRules: rules,
	}
}

func TestTestTarget(t *testing.T) {
	prod := BranchDeployRule{Pattern: "main", DockerNetwork: "prod_net", HostnameTemplate: "app.example.com", Port: 80}
	preview := BranchDeployRule{Pattern: "feature/*", DockerNetwork: "qa_net", HostnameTemplate: "{branch}.example.com", Port: 80}
	previewOnProd := BranchDeployRule{Pattern: "feature/*", DockerNetwork: "prod_net", HostnameTemplate: "{branch}.example.com", Port: 80}
	previewOverridden := BranchDeployRule{Pattern: "feature/*", DockerNetwork: "prod_net", HostnameTemplate: "{branch}.example.com", Port: 80, Overrides: map[string]string{"DATABASE_URL": "qa"}}
	previewNoHost := BranchDeployRule{Pattern: "feature/*", DockerNetwork: "qa_net"}
	shared := BranchDeployRule{Pattern: "dev", DockerNetwork: "qa_net", NameSuffix: "dev", HostnameTemplate: "qa.example.com", Port: 80}
	sharedOnProd := BranchDeployRule{Pattern: "staging", DockerNetwork: "prod_net", NameSuffix: "staging", HostnameTemplate: "staging.example.com", Port: 80}
	inPlace := BranchDeployRule{Pattern: "hotfix", DockerNetwork: "qa_net", HostnameTemplate: "hotfix.example.com", Port: 80}
	defaultClone := BranchDeployRule{Pattern: "main", DockerNetwork: "qa_net", NameSuffix: "copy", HostnameTemplate: "copy.example.com", Port: 80}
	feature := []BranchRef{{Owner: "Acme", Repo: "APP", Branch: "feature/dot.test"}}

	tests := []struct {
		name     string
		stack    *Stack
		branches []BranchRef
		want     TestTarget
	}{
		{"preview on its own network", newTargetStack(prod, preview), feature, TestTarget{URL: "https://dot-test.example.com", Kind: TestTargetPreview, Branch: "feature/dot.test"}},
		{"preview on production's network with overrides", newTargetStack(previewOverridden), feature, TestTarget{URL: "https://dot-test.example.com", Kind: TestTargetPreview, Branch: "feature/dot.test"}},
		{"preview sharing production's network falls back to shared", newTargetStack(previewOnProd, shared), feature, TestTarget{URL: "https://qa.example.com", Kind: TestTargetShared, Branch: "dev"}},
		{"preview without a hostname falls back to shared", newTargetStack(previewNoHost, shared), feature, TestTarget{URL: "https://qa.example.com", Kind: TestTargetShared, Branch: "dev"}},
		{"branch from another repository gets the shared environment", newTargetStack(preview, shared), []BranchRef{{Owner: "acme", Repo: "other", Branch: "feature/x"}}, TestTarget{URL: "https://qa.example.com", Kind: TestTargetShared, Branch: "dev"}},
		{"no linked branch gets the shared environment", newTargetStack(prod, shared), nil, TestTarget{URL: "https://qa.example.com", Kind: TestTargetShared, Branch: "dev"}},
		{"the default branch is never offered", newTargetStack(prod, preview), []BranchRef{{Owner: "acme", Repo: "app", Branch: "main"}}, TestTarget{}},
		{"a clone of the default branch is still production", newTargetStack(defaultClone), nil, TestTarget{}},
		{"shared rule on production's network is never offered", newTargetStack(prod, sharedOnProd), nil, TestTarget{}},
		{"an in-place rule redeploys production", newTargetStack(inPlace), []BranchRef{{Owner: "acme", Repo: "app", Branch: "hotfix"}}, TestTarget{}},
		{"no rules", newTargetStack(), feature, TestTarget{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, testTarget([]*Stack{tt.stack}, tt.branches))
		})
	}
}

func TestTestTarget_ComposeStackUsesItsProjectNetwork(t *testing.T) {
	stack := newTargetStack(BranchDeployRule{Pattern: "feature/*", DockerNetwork: "web_default", HostnameTemplate: "{branch}.example.com", Port: 80})
	stack.Strategy = StrategyCompose

	assert.Equal(t, TestTarget{}, testTarget([]*Stack{stack}, []BranchRef{{Owner: "acme", Repo: "app", Branch: "feature/a"}}))
}

func TestTestTarget_ImageOnlyStackDefaultsToMain(t *testing.T) {
	stack := newTargetStack(BranchDeployRule{Pattern: "main", DockerNetwork: "qa_net", NameSuffix: "qa", HostnameTemplate: "qa.example.com", Port: 80})
	stack.BuildSource = nil

	assert.Equal(t, TestTarget{}, testTarget([]*Stack{stack}, nil))
}

func TestResolveTestTarget_SkipsBranchDeployments(t *testing.T) {
	stacks := newFakeStackRepo()
	base := newTargetStack(BranchDeployRule{Pattern: "feature/*", DockerNetwork: "qa_net", HostnameTemplate: "{branch}.example.com", Port: 80})
	require.NoError(t, stacks.Create(t.Context(), base))
	clone := newTargetStack(BranchDeployRule{Pattern: "dev", DockerNetwork: "qa_net", NameSuffix: "dev", HostnameTemplate: "clone.example.com", Port: 80})
	clone.ID, clone.DerivedFrom = "s2", "s1"
	require.NoError(t, stacks.Create(t.Context(), clone))
	s := NewService(newFakeRepo(), stacks, newFakeContainerRepo(), newFakeProjects())

	got, err := s.ResolveTestTarget(t.Context(), "p1", []BranchRef{{Owner: "acme", Repo: "app", Branch: "feature/a"}})

	require.NoError(t, err)
	assert.Equal(t, TestTarget{URL: "https://a.example.com", Kind: TestTargetPreview, Branch: "feature/a"}, got)
}

func TestResolveTestTarget_ListFails_ReturnsError(t *testing.T) {
	stacks := newFakeStackRepo()
	stacks.listErr = errors.New("disk gone")
	s := NewService(newFakeRepo(), stacks, newFakeContainerRepo(), newFakeProjects())

	_, err := s.ResolveTestTarget(t.Context(), "p1", nil)

	require.ErrorContains(t, err, "disk gone")
}
