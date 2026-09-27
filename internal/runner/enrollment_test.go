package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
)

var enrollNow = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

// newEnrollService wires enrollment over fakes, with the admin gate newUpgradeService uses and a fixed clock.
func newEnrollService(repo *fakeRunnerRepo, dispatch *fakeDispatch) *Service {
	svc := NewService(repo, dispatch).
		WithMachines(repo.machines).
		WithInstall(InstallConfig{Settings: &fakeSettingsReader{url: "https://nexul.example.com/"}}).
		WithAdminGate(fakeAdminGate{admins: map[string]bool{"admin-1": true}}).
		WithBus(newFakeBus())
	svc.now = func() time.Time { return enrollNow }
	return svc
}

func TestService_CreateEnrollment_Refusals(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		settings SettingsReader
		runner   string
		want     error
	}{
		{"no actor", context.Background(), nil, "build-box", apperrs.ErrUnauthorized},
		{"not an instance admin", asMember(), nil, "build-box", apperrs.ErrForbidden},
		{"no instance url yet", asAdmin(), &fakeSettingsReader{}, "build-box", apperrs.ErrConflict},
		{"settings lookup fails", asAdmin(), &fakeSettingsReader{err: assert.AnError}, "build-box", assert.AnError},
		{"empty name", asAdmin(), nil, " ", apperrs.ErrInvalid},
		{"name with capitals", asAdmin(), nil, "Build", apperrs.ErrInvalid},
		{"name with a leading dash", asAdmin(), nil, "-box", apperrs.ErrInvalid},
		{"name longer than 32", asAdmin(), nil, strings.Repeat("a", 33), apperrs.ErrInvalid},
		{"name already enrolled", asAdmin(), nil, "taken", apperrs.ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRunnerRepo()
			repo.enrolled("r-1", "taken", "")
			svc := newEnrollService(repo, &fakeDispatch{})
			if tt.settings != nil {
				svc.install.Settings = tt.settings
			}
			_, err := svc.CreateEnrollment(tt.ctx, tt.runner, "")
			require.ErrorIs(t, err, tt.want)
			assert.Empty(t, repo.codes, "a refused enrollment stores no code")
		})
	}
}

func TestService_CreateEnrollment_RepoFailureIsReturned(t *testing.T) {
	repo := newFakeRunnerRepo()
	repo.createErr = assert.AnError
	_, err := newEnrollService(repo, &fakeDispatch{}).CreateEnrollment(asAdmin(), "build-box", "")
	require.ErrorIs(t, err, assert.AnError)
}

func TestService_CreateEnrollment_RendersTheInstallCommands(t *testing.T) {
	t.Run("a release build pins the installer to its own version", func(t *testing.T) {
		withVersion(t, "v0.3.0-beta-012")
		e, err := newEnrollService(newFakeRunnerRepo(), &fakeDispatch{}).CreateEnrollment(asAdmin(), " build-box ", "prod")
		require.NoError(t, err)

		assert.True(t, strings.HasPrefix(e.Code, "nxe_"))
		assert.Equal(t, enrollNow.Add(time.Hour), e.ExpiresAt)
		assert.Equal(t, "curl -fsSL https://nexul.io/runner.sh | NEXUL_VERSION=v0.3.0-beta-012 sh -s -- --server https://nexul.example.com --name build-box --code "+e.Code, e.Commands.Unix)
		assert.Equal(t, "$env:NEXUL_VERSION='v0.3.0-beta-012'; & ([scriptblock]::Create((irm https://nexul.io/runner.ps1))) --server https://nexul.example.com --name build-box --code "+e.Code, e.Commands.Windows)
	})

	t.Run("a dev build leaves the version to the installer", func(t *testing.T) {
		withVersion(t, "dev")
		e, err := newEnrollService(newFakeRunnerRepo(), &fakeDispatch{}).CreateEnrollment(asAdmin(), "build-box", "")
		require.NoError(t, err)
		assert.Equal(t, "curl -fsSL https://nexul.io/runner.sh | sh -s -- --server https://nexul.example.com --name build-box --code "+e.Code, e.Commands.Unix)
		assert.Equal(t, "& ([scriptblock]::Create((irm https://nexul.io/runner.ps1))) --server https://nexul.example.com --name build-box --code "+e.Code, e.Commands.Windows)
	})

	t.Run("the stored code is only a hash, bound to the name and machine", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		e, err := newEnrollService(repo, &fakeDispatch{}).CreateEnrollment(asAdmin(), "build-box", " prod ")
		require.NoError(t, err)
		stored, ok := repo.codes[hostcred.Hash(e.Code)]
		require.True(t, ok)
		assert.Equal(t, "build-box", stored.Name)
		assert.Equal(t, "prod", stored.Machine)
	})
}

func mustEnrollment(t *testing.T, svc *Service, name, machine string) string {
	t.Helper()
	e, err := svc.CreateEnrollment(asAdmin(), name, machine)
	require.NoError(t, err)
	return e.Code
}

func TestService_Enroll_Refusals(t *testing.T) {
	t.Run("an unknown code", func(t *testing.T) {
		svc := newEnrollService(newFakeRunnerRepo(), &fakeDispatch{})
		_, err := svc.Enroll(context.Background(), EnrollRequest{Code: "nxe_nope", Name: "build-box"})
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
		var coded *apperrs.Coded
		require.ErrorAs(t, err, &coded)
		assert.Equal(t, "invalid_code", coded.Code)
	})

	t.Run("an expired code", func(t *testing.T) {
		svc := newEnrollService(newFakeRunnerRepo(), &fakeDispatch{})
		code := mustEnrollment(t, svc, "build-box", "")
		svc.now = func() time.Time { return enrollNow.Add(time.Hour) }
		_, err := svc.Enroll(context.Background(), EnrollRequest{Code: code, Name: "build-box"})
		require.ErrorIs(t, err, errInvalidCode)
	})

	t.Run("a used code", func(t *testing.T) {
		svc := newEnrollService(newFakeRunnerRepo(), &fakeDispatch{})
		code := mustEnrollment(t, svc, "build-box", "")
		_, err := svc.Enroll(context.Background(), EnrollRequest{Code: code, Name: "build-box"})
		require.NoError(t, err)
		_, err = svc.Enroll(context.Background(), EnrollRequest{Code: code, Name: "build-box"})
		require.ErrorIs(t, err, errInvalidCode)
	})

	t.Run("another runner's name keeps the code usable", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		svc := newEnrollService(repo, &fakeDispatch{})
		code := mustEnrollment(t, svc, "build-box", "")
		_, err := svc.Enroll(context.Background(), EnrollRequest{Code: code, Name: "other"})
		require.ErrorIs(t, err, apperrs.ErrConflict)
		var coded *apperrs.Coded
		require.ErrorAs(t, err, &coded)
		assert.Equal(t, "name_mismatch", coded.Code)
		assert.Len(t, repo.codes, 1)
	})

	t.Run("a name taken since the code was minted is a conflict", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		svc := newEnrollService(repo, &fakeDispatch{})
		code := mustEnrollment(t, svc, "build-box", "")
		repo.enrolled("r-9", "build-box", "")
		_, err := svc.Enroll(context.Background(), EnrollRequest{Code: code, Name: "build-box"})
		require.ErrorIs(t, err, apperrs.ErrConflict, "the name was taken between the code and its use")
	})
}

func TestService_Enroll_CreatesTheRunnerOnItsMachine(t *testing.T) {
	t.Run("without a machine the runner gets its own, seeded with its stack root", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		svc := newEnrollService(repo, &fakeDispatch{})
		code := mustEnrollment(t, svc, "build-box", "")

		got, err := svc.Enroll(context.Background(), EnrollRequest{Code: code, Name: "build-box", Version: "v0.3.0", StackRoot: "/srv/nexul"})
		require.NoError(t, err)

		assert.Equal(t, "build-box", got.Name)
		assert.Equal(t, "build-box", got.Machine)
		assert.True(t, strings.HasPrefix(got.Credential, "nxr_"))
		r, err := repo.GetByID(context.Background(), got.ID)
		require.NoError(t, err)
		assert.Equal(t, "v0.3.0", r.Version)
		m, err := repo.machines.Get(context.Background(), r.MachineID)
		require.NoError(t, err)
		assert.Equal(t, "build-box", m.Name)
		assert.Equal(t, "/srv/nexul", m.StackRoot)
		cred, err := authenticate(context.Background(), repo, got.Credential)
		require.NoError(t, err)
		assert.Equal(t, got.ID, cred.RunnerID)
	})

	t.Run("the reported hostname files the runner when the code names no machine", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		svc := newEnrollService(repo, &fakeDispatch{})
		code := mustEnrollment(t, svc, "second", "")

		got, err := svc.Enroll(context.Background(), EnrollRequest{Code: code, Name: "second", Machine: " box-1 "})
		require.NoError(t, err)

		assert.Equal(t, "box-1", got.Machine)
	})

	t.Run("the code's machine wins over the reported hostname", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		svc := newEnrollService(repo, &fakeDispatch{})
		code := mustEnrollment(t, svc, "second", "prod")

		got, err := svc.Enroll(context.Background(), EnrollRequest{Code: code, Name: "second", Machine: "box-1"})
		require.NoError(t, err)

		assert.Equal(t, "prod", got.Machine)
	})

	t.Run("a named machine is joined, keeping its own stack root", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		repo.enrolled("r-1", "first", "prod")
		svc := newEnrollService(repo, &fakeDispatch{})
		code := mustEnrollment(t, svc, "second", "prod")

		got, err := svc.Enroll(context.Background(), EnrollRequest{Code: code, Name: "second", StackRoot: "/elsewhere"})
		require.NoError(t, err)

		assert.Equal(t, "prod", got.Machine)
		r, err := repo.GetByID(context.Background(), got.ID)
		require.NoError(t, err)
		assert.Equal(t, "m-prod", r.MachineID)
		m, err := repo.machines.Get(context.Background(), "m-prod")
		require.NoError(t, err)
		assert.Equal(t, defaultStackRoot, m.StackRoot)
	})
}

func TestService_WriteInstanceEnrollment(t *testing.T) {
	t.Run("writes a 24 hour code for the bundled runner while it is not enrolled", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		dir := filepath.Join(t.TempDir(), "enroll")
		svc := newEnrollService(repo, &fakeDispatch{}).WithEnrollDir(dir)

		require.NoError(t, svc.WriteInstanceEnrollment(context.Background()))

		path := filepath.Join(dir, "runner-instance")
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		code, err := os.ReadFile(path)
		require.NoError(t, err)
		stored, ok := repo.codes[hostcred.Hash(string(code))]
		require.True(t, ok)
		assert.Equal(t, "instance", stored.Name)
		assert.Equal(t, enrollNow.Add(24*time.Hour), stored.ExpiresAt)

		got, err := svc.Enroll(context.Background(), EnrollRequest{Code: string(code), Name: "instance"})
		require.NoError(t, err)
		assert.Equal(t, "instance", got.Machine)
		_, err = os.Stat(path)
		assert.ErrorIs(t, err, os.ErrNotExist, "enrolling the bundled runner deletes its code file")
	})

	t.Run("deletes a leftover file once the bundled runner is enrolled", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		repo.enrolled("r-i", "instance", "instance")
		dir := t.TempDir()
		path := filepath.Join(dir, "runner-instance")
		require.NoError(t, os.WriteFile(path, []byte("nxe_old"), 0o600))

		require.NoError(t, newEnrollService(repo, &fakeDispatch{}).WithEnrollDir(dir).WriteInstanceEnrollment(context.Background()))

		_, err := os.Stat(path)
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Empty(t, repo.codes)
	})

	t.Run("an unwritable directory fails", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "not-a-dir")
		require.NoError(t, os.WriteFile(file, nil, 0o600))
		err := newEnrollService(newFakeRunnerRepo(), &fakeDispatch{}).WithEnrollDir(file).WriteInstanceEnrollment(context.Background())
		require.Error(t, err)
	})
}

func TestService_RemoveRunner(t *testing.T) {
	t.Run("refused for anyone but an instance admin", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		repo.enrolled("r-1", "build-box", "")
		dispatch := &fakeDispatch{}
		err := newEnrollService(repo, dispatch).RemoveRunner(asMember(), "r-1")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		assert.Empty(t, dispatch.uninstalled)
		_, err = repo.GetByID(context.Background(), "r-1")
		require.NoError(t, err)
	})

	t.Run("an unknown runner is not found", func(t *testing.T) {
		dispatch := &fakeDispatch{}
		err := newEnrollService(newFakeRunnerRepo(), dispatch).RemoveRunner(asAdmin(), "ghost")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		assert.Empty(t, dispatch.uninstalled)
	})

	t.Run("revokes, deletes, tells the runner and announces it", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		cred := repo.enrolled("r-1", "build-box", "")
		dispatch := &fakeDispatch{}
		bus := newFakeBus()
		svc := newEnrollService(repo, dispatch).WithBus(bus)

		require.NoError(t, svc.RemoveRunner(asAdmin(), "r-1"))

		_, err := repo.GetByID(context.Background(), "r-1")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		got, err := authenticate(context.Background(), repo, cred)
		require.NoError(t, err)
		assert.True(t, got.Revoked)
		assert.Equal(t, []string{"r-1"}, dispatch.uninstalled)
		evs := bus.topicEvents(TopicRunnerDisconnected)
		require.Len(t, evs, 1)
		assert.Equal(t, RunnerDisconnectedEvent{RunnerID: "r-1", Reason: "removed"}, decodeEvent[RunnerDisconnectedEvent](t, evs[0]))
	})
}

func TestService_RemoveSelf(t *testing.T) {
	t.Run("an unknown credential is unauthorized", func(t *testing.T) {
		err := newEnrollService(newFakeRunnerRepo(), &fakeDispatch{}).RemoveSelf(context.Background(), "nxr_nope")
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})

	t.Run("an empty credential is unauthorized", func(t *testing.T) {
		err := newEnrollService(newFakeRunnerRepo(), &fakeDispatch{}).RemoveSelf(context.Background(), "")
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})

	t.Run("a credential lookup failure surfaces", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		repo.credErr = assert.AnError
		err := newEnrollService(repo, &fakeDispatch{}).RemoveSelf(context.Background(), "nxr_x")
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("removes the credential's runner, and again is a no-op", func(t *testing.T) {
		repo := newFakeRunnerRepo()
		cred := repo.enrolled("r-1", "build-box", "")
		dispatch := &fakeDispatch{}
		svc := newEnrollService(repo, dispatch)

		require.NoError(t, svc.RemoveSelf(context.Background(), cred))
		require.NoError(t, svc.RemoveSelf(context.Background(), cred), "a removed runner has nothing left to remove")

		_, err := repo.GetByID(context.Background(), "r-1")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		assert.Equal(t, []string{"r-1"}, dispatch.uninstalled)
	})
}

// enrollmentServer mounts the session routes as an instance admin next to the public routes.
func enrollmentServer(t *testing.T, svc *Service) *httptest.Server {
	t.Helper()
	routes, public := NewHTTPHandler(svc).Routes(), NewHTTPHandler(svc).PublicRoutes()
	mux := http.NewServeMux()
	mux.Handle("/api/runners/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routes.ServeHTTP(w, r.WithContext(asAdmin()))
	}))
	mux.Handle("POST /api/runners/enroll", public)
	mux.Handle("POST /api/runners/self/remove", public)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func postJSON(t *testing.T, url, auth string, body any) (*http.Response, map[string]any) {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(b)))
	require.NoError(t, err)
	if auth != "" {
		req.Header.Set("Authorization", "Bearer "+auth)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() }) // read in full below; close errors are irrelevant to the test
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out) // a 204 has no body to decode
	return resp, out
}

func TestHTTPHandler_Enrollment(t *testing.T) {
	withVersion(t, "dev")
	repo := newFakeRunnerRepo()
	srv := enrollmentServer(t, newEnrollService(repo, &fakeDispatch{}))

	resp, _ := postJSON(t, srv.URL+"/api/runners/enrollments", "", map[string]string{"name": "Bad Name"})
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp, _ = postJSON(t, srv.URL+"/api/runners/enrollments", "", "not an object")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp, body := postJSON(t, srv.URL+"/api/runners/enrollments", "", map[string]string{"name": "build-box"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	code, _ := body["code"].(string)
	require.NotEmpty(t, code)
	commands, _ := body["commands"].(map[string]any)
	assert.Contains(t, commands["unix"], "--code "+code)
	assert.Contains(t, commands["windows"], "--code "+code)
	assert.NotEmpty(t, body["expires_at"])

	resp, _ = postJSON(t, srv.URL+"/api/runners/enroll", "", "not an object")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp, body = postJSON(t, srv.URL+"/api/runners/enroll", "", EnrollRequest{Code: code, Name: "other"})
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.Equal(t, "name_mismatch", body["code"])

	resp, body = postJSON(t, srv.URL+"/api/runners/enroll", "", EnrollRequest{Code: code, Name: "build-box", OS: "linux", Arch: "amd64"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "build-box", body["name"])
	assert.Equal(t, "build-box", body["machine"])
	credential, _ := body["credential"].(string)
	id, _ := body["id"].(string)
	require.NotEmpty(t, credential)

	resp, body = postJSON(t, srv.URL+"/api/runners/enroll", "", EnrollRequest{Code: code, Name: "build-box"})
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "invalid_code", body["code"])

	resp, _ = postJSON(t, srv.URL+"/api/runners/self/remove", "nxr_nope", nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp, _ = postJSON(t, srv.URL+"/api/runners/self/remove", credential, nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	_, err := repo.GetByID(context.Background(), id)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestHTTPHandler_RemoveRunner(t *testing.T) {
	repo := newFakeRunnerRepo()
	repo.enrolled("r-1", "build-box", "")
	srv := enrollmentServer(t, newEnrollService(repo, &fakeDispatch{}))

	del := func(id string) int {
		req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/runners/"+id, nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode
	}
	assert.Equal(t, http.StatusNotFound, del("ghost"))
	assert.Equal(t, http.StatusNoContent, del("r-1"))
	assert.Equal(t, http.StatusNotFound, del("r-1"))
}
