package deploy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
)

func newDeployHandler(t *testing.T) *Handler {
	t.Helper()
	stacks := newFakeStackRepo()
	projects := newFakeProjects()
	projects.exists["proj-1"] = true
	stack := validStack()
	requireStack(t, stacks, stack)
	s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
	return NewHandler(s)
}

func decodeDeploy(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), v))
}

func withActor(r *http.Request) *http.Request {
	return r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: "user-1", CanCreateWorkspace: true}))
}

func TestDeployRoutes_Deploy_ValidationError(t *testing.T) {
	h := newDeployHandler(t).Routes()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/deploys",
		strings.NewReader(`{"stack_id":"","image":""}`)))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeployRoutes_Deploy_PublishOnly(t *testing.T) {
	h := newDeployHandler(t).Routes()
	body := `{"stack_id":"svc-1","image":"ghcr.io/x/api"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withActor(httptest.NewRequest(http.MethodPost, "/api/deploys", strings.NewReader(body))))
	assert.Equal(t, http.StatusCreated, rec.Code)
	var d Deploy
	decodeDeploy(t, rec, &d)
	assert.Equal(t, "svc-1", d.StackID)
	assert.Equal(t, "api", d.Service)
	assert.Equal(t, StatusPending, d.Status)
	assert.Equal(t, "user-1", d.TriggeredBy, "provenance comes from the authenticated actor")
	assert.NotEmpty(t, d.ID)
}

func TestDeployRoutes_Deploy_ServiceIDAlias(t *testing.T) {
	h := newDeployHandler(t).Routes()
	body := `{"service_id":"svc-1","image":"ghcr.io/x/api"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/deploys", strings.NewReader(body)))
	assert.Equal(t, http.StatusCreated, rec.Code)
	var d Deploy
	decodeDeploy(t, rec, &d)
	assert.Equal(t, "svc-1", d.StackID, "the deprecated service_id body field still resolves the stack")
}

func TestDeployRoutes_List_Empty(t *testing.T) {
	h := newDeployHandler(t).Routes()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/deploys", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	var ds []*Deploy
	decodeDeploy(t, rec, &ds)
	assert.Empty(t, ds)
}

func TestDeployRoutes_Get_NotFound(t *testing.T) {
	h := newDeployHandler(t).Routes()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/deploys/nope", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeployRoutes_ListByStatus_Invalid(t *testing.T) {
	h := newDeployHandler(t).Routes()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/deploys?status=bogus", nil))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeployRoutes_Cancel(t *testing.T) {
	h := newDeployHandler(t).Routes()

	t.Run("queued deploy returns 202", func(t *testing.T) {
		repo := newFakeRepo()
		seedDeploy(t, repo, &Deploy{ID: "d1", Service: "api", Status: StatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		svc := newTestService(repo, newFakeBus())
		h := NewHandler(svc).Routes()

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/deploys/d1/cancel", nil))
		assert.Equal(t, http.StatusAccepted, rec.Code)
		var body map[string]string
		decodeDeploy(t, rec, &body)
		assert.Equal(t, "d1", body["id"])
		assert.Equal(t, "cancelling", body["status"])
	})

	t.Run("terminal deploy is a conflict", func(t *testing.T) {
		repo := newFakeRepo()
		seedDeploy(t, repo, &Deploy{ID: "d1", Service: "api", Status: StatusHealthy, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		h := NewHandler(newTestService(repo, newFakeBus())).Routes()

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/deploys/d1/cancel", nil))
		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("unknown deploy is 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/deploys/nope/cancel", nil))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestStackRoutes(t *testing.T) {
	t.Run("create returns 201 and the stack", func(t *testing.T) {
		h := newDeployHandler(t).Routes()
		body := `{"project_id":"proj-1","name":"web","machine":"10.0.0.1:22","strategy":"compose","compose_path":"/srv/web/docker-compose.yml"}`
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, withActor(httptest.NewRequest(http.MethodPost, "/api/stacks", strings.NewReader(body))))
		assert.Equal(t, http.StatusCreated, rec.Code)
		var stack Stack
		decodeDeploy(t, rec, &stack)
		assert.Equal(t, "web", stack.Name)
		assert.Equal(t, "web", stack.Slug)
		assert.NotEmpty(t, stack.ID)
	})
	t.Run("create links the repository and enqueues the first deploy when asked", func(t *testing.T) {
		h := newDeployHandler(t).Routes()
		body := `{"project_id":"proj-1","name":"hello","machine":"10.0.0.1:22","strategy":"run","docker_network":"app","build_source":{"repo_owner":"onik97","repo_name":"hello","branch":"main","dockerfile":"Dockerfile"},"link_repository":true,"deploy":true}`
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, withActor(httptest.NewRequest(http.MethodPost, "/api/stacks", strings.NewReader(body))))
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var out struct {
			Stack
			Deploy *Deploy `json:"deploy"`
		}
		decodeDeploy(t, rec, &out)
		assert.Equal(t, "hello", out.Slug)
		require.NotNil(t, out.Deploy)
		assert.Equal(t, out.ID, out.Deploy.StackID)
		assert.Equal(t, KindBuild, out.Deploy.Kind)
	})
	t.Run("create with declared services", func(t *testing.T) {
		h := newDeployHandler(t).Routes()
		body := `{"project_id":"proj-1","name":"stack2","machine":"10.0.0.1:22","strategy":"compose","compose_path":"docker-compose.yml","declared":{"web":{"image":"nginx"}}}`
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/stacks", strings.NewReader(body)))
		require.Equal(t, http.StatusCreated, rec.Code)
		var stack Stack
		decodeDeploy(t, rec, &stack)

		svcsRec := httptest.NewRecorder()
		h.ServeHTTP(svcsRec, httptest.NewRequest(http.MethodGet, "/api/stacks/"+stack.ID+"/services", nil))
		assert.Equal(t, http.StatusOK, svcsRec.Code)
		var svcs []*Container
		decodeDeploy(t, svcsRec, &svcs)
		require.Len(t, svcs, 1)
		assert.Equal(t, "web", svcs[0].Name)
	})
	t.Run("create duplicate slug is a conflict", func(t *testing.T) {
		h := newDeployHandler(t).Routes()
		body := `{"project_id":"proj-1","name":"api","machine":"10.0.0.1:22","strategy":"compose","compose_path":"/srv/api/docker-compose.yml"}`
		for range 2 {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, withActor(httptest.NewRequest(http.MethodPost, "/api/stacks", strings.NewReader(body))))
			if rec.Code == http.StatusConflict {
				return
			}
		}
		t.Fatal("expected a duplicate-slug conflict")
	})
	t.Run("list without a project spans every project", func(t *testing.T) {
		rec := httptest.NewRecorder()
		newDeployHandler(t).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"id":"svc-1"`)
	})
	t.Run("the /api/services alias serves the same handlers", func(t *testing.T) {
		rec := httptest.NewRecorder()
		newDeployHandler(t).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/services", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"id":"svc-1"`)
	})
	t.Run("get not found is 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		newDeployHandler(t).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks/nope", nil))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("get attaches branch deployments to a base stack, and lists stay clean", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		requireStack(t, stacks, validStack())
		derived := validStack()
		derived.ID = "svc-2"
		derived.Name = "api-feature-x"
		derived.Slug = "api-feature-x"
		derived.DerivedFrom = "svc-1"
		derived.Branch = "feature/x"
		requireStack(t, stacks, derived)
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		h := NewHandler(s).Routes()

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks/svc-1", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
		var got stackResponse
		decodeDeploy(t, rec, &got)
		require.Len(t, got.BranchDeployments, 1)
		assert.Equal(t, "api-feature-x", got.BranchDeployments[0].Name)

		listRec := httptest.NewRecorder()
		h.ServeHTTP(listRec, httptest.NewRequest(http.MethodGet, "/api/stacks?project_id=proj-1", nil))
		var list []*Stack
		decodeDeploy(t, listRec, &list)
		require.Len(t, list, 1, "the derived clone stays off the top-level list")
		assert.Equal(t, "api", list[0].Name)
	})
	t.Run("update requires body id and returns the updated stack", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		requireStack(t, stacks, validStack())
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		h := NewHandler(s).Routes()

		body := `{"project_id":"proj-1","name":"api-v2","machine":"10.0.0.1:22","strategy":"compose","compose_path":"/srv/api/docker-compose.yml"}`
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, withActor(httptest.NewRequest(http.MethodPatch, "/api/stacks/svc-1", strings.NewReader(body))))
		assert.Equal(t, http.StatusOK, rec.Code)
		var got Stack
		decodeDeploy(t, rec, &got)
		assert.Equal(t, "api-v2", got.Name)
	})
	t.Run("delete returns deleted", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		requireStack(t, stacks, validStack())
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		h := NewHandler(s).Routes()

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, withActor(httptest.NewRequest(http.MethodDelete, "/api/stacks/svc-1", nil)))
		assert.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("rollback triggers a deploy", func(t *testing.T) {
		repo := newFakeRepo()
		now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", StackID: "svc-1", Service: "api", Image: "img:v1", Status: StatusHealthy, CreatedAt: now, UpdatedAt: now}))
		s := newTestService(repo, newFakeBus())
		h := NewHandler(s).Routes()

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, withActor(httptest.NewRequest(http.MethodPost, "/api/stacks/svc-1/rollback", nil)))
		assert.Equal(t, http.StatusCreated, rec.Code)
		var d Deploy
		decodeDeploy(t, rec, &d)
		assert.Equal(t, "img:v1", d.Image)
		assert.Equal(t, "user-1", d.TriggeredBy)
	})
	t.Run("list stack deploys returns history", func(t *testing.T) {
		repo := newFakeRepo()
		now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", StackID: "svc-1", Service: "api", Image: "img:v1", Status: StatusHealthy, CreatedAt: now, UpdatedAt: now}))
		s := newTestService(repo, newFakeBus())
		h := NewHandler(s).Routes()

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks/svc-1/deploys", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
		var ds []*Deploy
		decodeDeploy(t, rec, &ds)
		require.Len(t, ds, 1)
		assert.Equal(t, "img:v1", ds[0].Image)
	})
	t.Run("list by stack id filter", func(t *testing.T) {
		repo := newFakeRepo()
		now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", StackID: "svc-1", Service: "api", Status: StatusHealthy, CreatedAt: now, UpdatedAt: now}))
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d2", StackID: "svc-2", Service: "web", Status: StatusFailed, CreatedAt: now, UpdatedAt: now}))
		s := newTestService(repo, newFakeBus())
		h := NewHandler(s).Routes()

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/deploys?stack_id=svc-1", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
		var ds []*Deploy
		decodeDeploy(t, rec, &ds)
		require.Len(t, ds, 1)
		assert.Equal(t, "d1", ds[0].ID)
	})
	t.Run("list by deprecated service_id filter", func(t *testing.T) {
		repo := newFakeRepo()
		now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
		require.NoError(t, repo.Create(context.Background(), &Deploy{ID: "d1", StackID: "svc-1", Service: "api", Status: StatusHealthy, CreatedAt: now, UpdatedAt: now}))
		s := newTestService(repo, newFakeBus())
		h := NewHandler(s).Routes()

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/deploys?service_id=svc-1", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
		var ds []*Deploy
		decodeDeploy(t, rec, &ds)
		require.Len(t, ds, 1)
		assert.Equal(t, "d1", ds[0].ID)
	})
	t.Run("stack not found is 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		newDeployHandler(t).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/stacks/nope", nil))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("update invalid body is 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		newDeployHandler(t).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, "/api/stacks/svc-1", strings.NewReader(`{bad`)))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("delete duplicate id path returns ok then not found", func(t *testing.T) {
		stacks := newFakeStackRepo()
		projects := newFakeProjects()
		projects.exists["proj-1"] = true
		requireStack(t, stacks, validStack())
		s := newTestServiceWith(newFakeRepo(), stacks, newFakeContainerRepo(), projects, newFakeBus())
		h := NewHandler(s).Routes()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, withActor(httptest.NewRequest(http.MethodDelete, "/api/stacks/svc-1", nil)))
		assert.Equal(t, http.StatusOK, rec.Code)
		rec2 := httptest.NewRecorder()
		h.ServeHTTP(rec2, withActor(httptest.NewRequest(http.MethodDelete, "/api/stacks/svc-1", nil)))
		assert.Equal(t, http.StatusNotFound, rec2.Code)
	})
}
