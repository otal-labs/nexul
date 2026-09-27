package runner

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func answer(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}
}

// firstStartConfig is a runner with no credential yet and an enrollment code file in a temp dir.
func firstStartConfig(t *testing.T) *RunnerConfig {
	t.Helper()
	dir := t.TempDir()
	return &RunnerConfig{
		ServerURL: "http://server:8080/", Name: "instance", StackRoot: "/data/nexul",
		CredentialFile: filepath.Join(dir, "state", "credential"), EnrollCodeFile: filepath.Join(dir, "runner-instance"),
		EnrollWait: time.Minute,
	}
}

func TestEnrollOnFirstStart_RefusedCode_FailsWithoutRetrying(t *testing.T) {
	cfg := firstStartConfig(t)
	require.NoError(t, os.WriteFile(cfg.EnrollCodeFile, []byte("nxe_used\n"), 0o600))
	calls := 0
	hc := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return answer(http.StatusUnauthorized, `{"message":"unauthorized","code":"invalid_code"}`), nil
	})}

	err := cfg.EnrollOnFirstStart(t.Context(), hc, "v1")
	require.ErrorIs(t, err, errEnrollRefused)
	assert.Contains(t, err.Error(), "invalid_code")
	assert.Equal(t, 1, calls)
	assert.NoFileExists(t, cfg.CredentialFile)
}

func TestEnrollOnFirstStart_AnswerWithoutCredential_Fails(t *testing.T) {
	cfg := firstStartConfig(t)
	require.NoError(t, os.WriteFile(cfg.EnrollCodeFile, []byte("nxe_abc"), 0o600))
	hc := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return answer(http.StatusCreated, `{"id":"r-1"}`), nil
	})}

	require.ErrorIs(t, cfg.EnrollOnFirstStart(t.Context(), hc, "v1"), errEnrollRefused)
}

func TestEnrollOnFirstStart_NothingToDo(t *testing.T) {
	hc := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("no request expected")
		return nil, nil
	})}
	enrolled := firstStartConfig(t)
	enrolled.Credential = "nxr_abc"
	require.NoError(t, enrolled.EnrollOnFirstStart(t.Context(), hc, "v1"))

	installed := firstStartConfig(t)
	installed.EnrollCodeFile = ""
	require.NoError(t, installed.EnrollOnFirstStart(t.Context(), hc, "v1"))
}

func TestEnrollOnFirstStart_WaitsForTheCodeAndTheServer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := firstStartConfig(t)
		var mu sync.Mutex
		var sent []map[string]string
		answers := []func() (*http.Response, error){
			func() (*http.Response, error) { return nil, errors.New("connection refused") },
			func() (*http.Response, error) { return answer(http.StatusServiceUnavailable, ""), nil },
			func() (*http.Response, error) { return answer(http.StatusCreated, `{"credential":"nxr_new"}`), nil },
		}
		hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				return nil, err
			}
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, "http://server:8080/api/runners/enroll", r.URL.String())
			sent = append(sent, body)
			next := answers[0]
			answers = answers[1:]
			return next()
		})}
		go func() {
			time.Sleep(5 * time.Second)
			_ = os.WriteFile(cfg.EnrollCodeFile, []byte("nxe_abc\n"), 0o600) // the server writing its code at boot
		}()

		require.NoError(t, cfg.EnrollOnFirstStart(t.Context(), hc, "v1"))
		assert.Equal(t, "nxr_new", cfg.Credential)
		data, err := os.ReadFile(cfg.CredentialFile)
		require.NoError(t, err)
		assert.Equal(t, "nxr_new\n", string(data))
		info, err := os.Stat(cfg.CredentialFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		require.Len(t, sent, 3)
		assert.Equal(t, "nxe_abc", sent[0]["code"])
		assert.Equal(t, "instance", sent[0]["name"])
		assert.Equal(t, "/data/nexul", sent[0]["stack_root"])
		assert.Equal(t, "v1", sent[0]["version"])
	})
}

func TestEnrollOnFirstStart_GivesUpAfterEnrollWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := firstStartConfig(t)
		cfg.EnrollWait = 10 * time.Second
		err := cfg.EnrollOnFirstStart(t.Context(), http.DefaultClient, "v1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), cfg.EnrollCodeFile)
	})
}

func TestLoadRunnerConfig_NoCredentialYetWithAnEnrollCode(t *testing.T) {
	t.Setenv("NEXUL_CREDENTIAL_FILE", filepath.Join(t.TempDir(), "credential"))
	t.Setenv("NEXUL_ENROLL_CODE_FILE", "/data/enroll/runner-instance")
	cfg, err := LoadRunnerConfig()
	require.NoError(t, err)
	assert.Empty(t, cfg.Credential)
	assert.Equal(t, "/data/enroll/runner-instance", cfg.EnrollCodeFile)
}
