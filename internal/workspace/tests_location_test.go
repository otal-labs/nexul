package workspace

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestAddRepo_Role(t *testing.T) {
	t.Run("empty role is an app repository and leaves the tests answer alone", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		require.NoError(t, s.AddRepo(t.Context(), "u-1", "p-1", "acme", "app", "", ""))
		assert.Equal(t, RepoRoleApp, repo.repos["p-1"][0].Role)
		assert.Equal(t, TestsLocationUnset, repo.projects["p-1"].TestsLocation)
	})
	t.Run("unknown role is invalid", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		err := s.AddRepo(t.Context(), "u-1", "p-1", "acme", "app", "", "docs")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		assert.Empty(t, repo.repos["p-1"])
	})
	t.Run("a tests repository records that tests live separately", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A", TestsLocation: TestsLocationSame}
		require.NoError(t, s.AddRepo(t.Context(), "u-1", "p-1", "acme", "e2e", "", RepoRoleTests))
		assert.Equal(t, RepoRoleTests, repo.repos["p-1"][0].Role)
		assert.Equal(t, TestsLocationSeparate, repo.projects["p-1"].TestsLocation)
	})
	t.Run("recording the tests answer can fail", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.updateErr = errors.New("db down")
		err := s.AddRepo(t.Context(), "u-1", "p-1", "acme", "e2e", "", RepoRoleTests)
		require.ErrorIs(t, err, repo.updateErr)
	})
	t.Run("a second tests repository does not rewrite an answer already separate", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A", TestsLocation: TestsLocationSeparate}
		repo.updateErr = errors.New("must not update")
		require.NoError(t, s.AddRepo(t.Context(), "u-1", "p-1", "acme", "e2e", "", RepoRoleTests))
	})
}

func TestSetTestsLocation(t *testing.T) {
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		_, err := s.SetTestsLocation(t.Context(), "u-1", "p-1", TestsLocationSame)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("unknown location is invalid", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		_, err := s.SetTestsLocation(t.Context(), "u-1", "p-1", "elsewhere")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("missing project is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.SetTestsLocation(t.Context(), "u-1", "nope", TestsLocationSame)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("update error propagates", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.updateErr = errors.New("db down")
		_, err := s.SetTestsLocation(t.Context(), "u-1", "p-1", TestsLocationSame)
		require.ErrorIs(t, err, repo.updateErr)
	})
	t.Run("records the answer, and an empty one withdraws it", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		p, err := s.SetTestsLocation(t.Context(), "u-1", "p-1", " same ")
		require.NoError(t, err)
		assert.Equal(t, TestsLocationSame, p.TestsLocation)
		assert.Equal(t, TestsLocationSame, repo.projects["p-1"].TestsLocation)

		_, err = s.SetTestsLocation(t.Context(), "u-1", "p-1", TestsLocationUnset)
		require.NoError(t, err)
		assert.Equal(t, TestsLocationUnset, repo.projects["p-1"].TestsLocation)
	})
}

func TestHandler_TestsLocation(t *testing.T) {
	t.Run("PUT records the answer", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		rec := do(t, h.Routes(), http.MethodPut, "/api/projects/p-1/tests-location", `{"tests_location":"separate"}`, "u-1")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, TestsLocationSeparate, decodeProject(t, rec).TestsLocation)
	})
	t.Run("bad body is invalid", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPut, "/api/projects/p-1/tests-location", `{`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("unknown location is invalid", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		rec := do(t, h.Routes(), http.MethodPut, "/api/projects/p-1/tests-location", `{"tests_location":"nope"}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("POST repos carries the role", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects/p-1/repos", `{"owner":"acme","name":"e2e","role":"tests"}`, "u-1")
		require.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, RepoRoleTests, repo.repos["p-1"][0].Role)
	})
}
