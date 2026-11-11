package runner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/synctest"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/secretref"
	"github.com/otal-labs/nexul/internal/platform/secretref/secretreftest"
)

func TestParseSecretCheckInterval(t *testing.T) {
	tests := []struct {
		raw     string
		want    time.Duration
		wantErr string
	}{
		{"", time.Hour, ""},
		{"off", 0, ""},
		{"OFF", 0, ""},
		{"5m", 5 * time.Minute, ""},
		{"90m", 90 * time.Minute, ""},
		{"4m59s", 0, "under the 5m0s minimum"},
		{"0s", 0, "under the 5m0s minimum"},
		{"-1h", 0, "under the 5m0s minimum"},
		{"hourly", 0, "is not a duration"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := ParseSecretCheckInterval(tt.raw)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.ErrorContains(t, err, "NEXUL_SECRET_CHECK_INTERVAL")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFingerprints_KeyIsMadeOnceHere_AndFilesAreOwnerOnly(t *testing.T) {
	dir := t.TempDir()
	store := NewFingerprints(dir)

	first, err := store.print("s3cret-pw")
	require.NoError(t, err)
	other, err := store.print("s3cret-pw-2")
	require.NoError(t, err)
	again, err := NewFingerprints(dir).print("s3cret-pw")
	require.NoError(t, err)

	assert.Equal(t, first, again, "a later runner process hashes with the key the first made")
	assert.NotEqual(t, first, other)
	assert.NotContains(t, first, "s3cret")
	elsewhere, err := NewFingerprints(t.TempDir()).print("s3cret-pw")
	require.NoError(t, err)
	assert.NotEqual(t, first, elsewhere, "another machine's key gives another fingerprint, so a stolen file proves nothing")

	require.NoError(t, store.save("api", stackFingerprints{DeployID: "d1", Vars: map[string]trackedVar{"DB": {Ref: "op://Acme/db/pw", Print: first}}}))
	if runtime.GOOS != "windows" {
		for _, name := range []string{"key", "api.json"} {
			info, err := os.Stat(filepath.Join(dir, "secret-check", name))
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), name)
		}
	}
	assert.Error(t, store.save("../escape", stackFingerprints{}), "a slug never names a path outside the directory")
}

// seedStack records a stack the way its healthy deploy would: each variable's reference and the fingerprint of the value.
func seedStack(t *testing.T, store *Fingerprints, slug, deployID string, refs, values map[string]string) {
	t.Helper()
	rec := stackFingerprints{DeployID: deployID, Vars: map[string]trackedVar{}}
	for name, ref := range refs {
		fp, err := store.print(values[name])
		require.NoError(t, err)
		rec.Vars[name] = trackedVar{Ref: ref, Print: fp}
	}
	require.NoError(t, store.save(slug, rec))
}

func newCheck(t *testing.T, every time.Duration, resolvers ...secretref.Resolver) (*SecretCheck, *Fingerprints) {
	t.Helper()
	store := NewFingerprints(t.TempDir())
	check := NewSecretCheck(store, func() secretref.Registry { return secretref.NewRegistry(resolvers...) }, every, testLogger())
	require.NotNil(t, check)
	return check, store
}

// A deploy's healthy result keeps what it resolved, a failed one leaves the last good record alone, a stack with no
// reference any more stops being checked, and a deleted stack's record goes with it.
func TestShellExecutor_KeepsFingerprintsOfAHealthyDeploy(t *testing.T) {
	dir := t.TempDir()
	fake := &secretreftest.Fake{Values: map[string]string{"op://Acme/db/password": "s3cret-pw"}}
	deploy := func(cmd *fakeCmd, id string, env map[string]string) {
		e := NewShellExecutor(cmd.run, ExecutorConfig{
			Registry:     func() secretref.Registry { return secretref.NewRegistry(fake) },
			Fingerprints: NewFingerprints(dir),
		}, testLogger())
		e.Deploy(context.Background(), DeployRequestedEvent{ID: id, Kind: RequestDeploy, StackSlug: "api", Image: "img:1", Strategy: "run", Env: env}, (&frameRecorder{}).send)
	}
	record := filepath.Join(dir, "secret-check", "api.json")
	refEnv := map[string]string{"DB_PASSWORD": "op://Acme/db/password", "PLAIN": "hello"}

	deploy(&fakeCmd{}, "d1", refEnv)
	raw, err := os.ReadFile(record)
	require.NoError(t, err)
	var rec stackFingerprints
	require.NoError(t, json.Unmarshal(raw, &rec))
	assert.Equal(t, "d1", rec.DeployID)
	assert.Equal(t, []string{"DB_PASSWORD"}, mapKeys(rec.Vars), "only the referenced variable is tracked")
	assert.NotContains(t, string(raw), "s3cret-pw", "the file holds a fingerprint, never the value")

	deploy(&fakeCmd{failNext: 1, err: assert.AnError}, "d2", refEnv)
	raw2, err := os.ReadFile(record)
	require.NoError(t, err)
	assert.JSONEq(t, string(raw), string(raw2), "a deploy that did not come up changes nothing")

	deploy(&fakeCmd{}, "d3", map[string]string{"PLAIN": "hello"})
	assert.NoFileExists(t, record, "a stack with no references left is not checked")

	deploy(&fakeCmd{}, "d4", refEnv)
	require.FileExists(t, record)
	NewShellExecutor((&fakeCmd{}).run, ExecutorConfig{Fingerprints: NewFingerprints(dir)}, testLogger()).
		Teardown(context.Background(), "api", "run", (&frameRecorder{}).send)
	assert.NoFileExists(t, record, "the record is removed with the stack")
}

func mapKeys[V any](m map[string]V) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// A changed value is reported once with names only, a value that changes again is reported again, and every stack
// (a preview included) is read in one batch per manager.
func TestSecretCheck_ReportsAChangedValueOnceWithNamesOnly(t *testing.T) {
	fake := &secretreftest.Fake{Values: map[string]string{
		"op://Acme/db/password": "pw-1", "op://Acme/web/token": "tok-1",
	}}
	check, store := newCheck(t, time.Hour, fake)
	refs := map[string]string{"DB_PASSWORD": "op://Acme/db/password", "TOKEN": "op://Acme/web/token"}
	values := map[string]string{"DB_PASSWORD": "pw-1", "TOKEN": "tok-1"}
	seedStack(t, store, "api", "d1", refs, values)
	seedStack(t, store, "api-feature-x", "d2", refs, values)

	check.check(t.Context())
	assert.Empty(t, check.take(), "nothing changed, nothing reported")
	require.Len(t, fake.Calls(), 1, "both stacks are read in one batch")
	assert.ElementsMatch(t, []string{"op://Acme/db/password", "op://Acme/web/token"}, fake.Calls()[0], "a reference both stacks hold is asked for once")

	fake.Values["op://Acme/db/password"] = "pw-2"
	check.check(t.Context())
	got := check.take()
	assert.Equal(t, []SecretChange{
		{Slug: "api", Variables: []string{"DB_PASSWORD"}},
		{Slug: "api-feature-x", Variables: []string{"DB_PASSWORD"}},
	}, got)
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	for _, leak := range []string{"op://", "Acme", "pw-1", "pw-2", "tok-1"} {
		assert.NotContains(t, string(raw), leak)
	}
	rec, err := store.load("api")
	require.NoError(t, err)
	assert.NotContains(t, string(raw), rec.Vars["DB_PASSWORD"].Print, "not even the fingerprint leaves the runner")

	check.check(t.Context())
	assert.Empty(t, check.take(), "the same change is not reported twice")

	fake.Values["op://Acme/db/password"] = "pw-3"
	check.check(t.Context())
	assert.Len(t, check.take(), 2, "a further change is reported again")
	assert.Len(t, fake.Calls(), 4, "one read per check, whatever the number of stacks")
}

func TestSecretCheck_NothingIsReadWhenThereIsNothingToCheck(t *testing.T) {
	fake := &secretreftest.Fake{Values: map[string]string{"op://Acme/db/password": "pw"}}
	store := NewFingerprints(t.TempDir())
	registry := func() secretref.Registry { return secretref.NewRegistry(fake) }

	assert.Nil(t, NewSecretCheck(store, registry, 0, testLogger()), "off is no check at all")
	assert.Nil(t, NewSecretCheck(nil, registry, time.Hour, testLogger()))

	check := NewSecretCheck(store, registry, time.Hour, testLogger())
	check.check(t.Context())
	assert.Empty(t, fake.Calls(), "no stack holds a reference, so there is no read")

	seedStack(t, store, "api", "d1", map[string]string{"DB": "op://Acme/db/password"}, map[string]string{"DB": "pw"})
	NewSecretCheck(store, func() secretref.Registry { return nil }, time.Hour, testLogger()).check(t.Context())
	assert.Empty(t, fake.Calls(), "a runner holding no manager reads nothing")
}

// A variable that cannot be read this time is left for the next check, and the others are still compared.
func TestSecretCheck_AnUnreadableVariableDoesNotHideTheRest(t *testing.T) {
	fake := &secretreftest.Fake{
		Values: map[string]string{"op://Acme/web/token": "tok-2"},
		Errs:   map[string]error{"op://Acme/db/password": secretref.ErrUnreachable},
	}
	check, store := newCheck(t, time.Hour, fake)
	seedStack(t, store, "api", "d1",
		map[string]string{"DB_PASSWORD": "op://Acme/db/password", "TOKEN": "op://Acme/web/token"},
		map[string]string{"DB_PASSWORD": "pw-1", "TOKEN": "tok-1"})

	check.check(t.Context())

	assert.Equal(t, []SecretChange{{Slug: "api", Variables: []string{"TOKEN"}}}, check.take())
}

// A stack deployed again while the check was reading carries its new deploy's values, so the stale finding is dropped.
func TestSecretCheck_ADeployDuringTheReadDropsTheStaleFinding(t *testing.T) {
	fake := &secretreftest.Fake{Values: map[string]string{"op://Acme/db/password": "pw-2"}}
	store := NewFingerprints(t.TempDir())
	resolver := redeployOnRead{Fake: fake, redeploy: func() {
		seedStack(t, store, "api", "d2", map[string]string{"DB": "op://Acme/db/password"}, map[string]string{"DB": "pw-2"})
	}}
	check := NewSecretCheck(store, func() secretref.Registry { return secretref.NewRegistry(resolver) }, time.Hour, testLogger())
	seedStack(t, store, "api", "d1", map[string]string{"DB": "op://Acme/db/password"}, map[string]string{"DB": "pw-1"})

	check.check(t.Context())

	assert.Empty(t, check.take())
}

type redeployOnRead struct {
	*secretreftest.Fake
	redeploy func()
}

func (r redeployOnRead) ResolveAll(ctx context.Context, refs []string) ([]secretref.Result, error) {
	out, err := r.Fake.ResolveAll(ctx, refs)
	r.redeploy()
	return out, err
}

// The check runs once per interval, on the fake clock, and ends with its context.
func TestSecretCheck_RunsOncePerInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := &secretreftest.Fake{Values: map[string]string{"op://Acme/db/password": "pw-2"}}
		check, store := newCheck(t, time.Hour, fake)
		seedStack(t, store, "api", "d1", map[string]string{"DB": "op://Acme/db/password"}, map[string]string{"DB": "pw-1"})
		ctx, cancel := context.WithCancel(t.Context())
		go check.run(ctx)

		time.Sleep(time.Hour - time.Second)
		synctest.Wait()
		assert.Empty(t, fake.Calls(), "the first check comes a full interval after start")

		time.Sleep(2 * time.Second)
		synctest.Wait()
		assert.Len(t, fake.Calls(), 1)
		assert.Len(t, check.take(), 1)

		time.Sleep(time.Hour)
		synctest.Wait()
		assert.Len(t, fake.Calls(), 2)

		cancel()
		synctest.Wait()
		time.Sleep(3 * time.Hour)
		synctest.Wait()
		assert.Len(t, fake.Calls(), 2, "no check after the context ends")
	})
}

// What the check found rides the runner's connection as one secrets_changed frame holding names and slugs only.
func TestClient_ReportsWhatTheSecretCheckFound(t *testing.T) {
	fake := &secretreftest.Fake{Values: map[string]string{"op://Acme/db/password": "pw-2"}}
	check, store := newCheck(t, time.Hour, fake)
	seedStack(t, store, "api", "d1", map[string]string{"DB_PASSWORD": "op://Acme/db/password"}, map[string]string{"DB_PASSWORD": "pw-1"})
	check.check(t.Context())

	got := make(chan json.RawMessage, 1)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		for {
			var raw json.RawMessage
			if err := wsjson.Read(ctx, conn, &raw); err != nil {
				return
			}
			if f, err := ParseFrame(raw); err == nil && f.Type == FrameSecretsChanged {
				got <- raw
				return
			}
		}
	})
	client := newTestClient(wsURL(srv), &fakeExecutor{})
	client.cfg.SecretCheck = check
	runClient(t, client)

	select {
	case raw := <-got:
		assert.JSONEq(t, `{"type":"secrets_changed","secret_changes":[{"slug":"api","variables":["DB_PASSWORD"]}]}`, string(raw))
	case <-time.After(5 * time.Second):
		t.Fatal("no secrets_changed frame")
	}
	assert.Empty(t, check.take(), "what was delivered is not kept for another connection")
}

func TestHost_SecretsChangedBecomesAnInternalBusEventStampedWithTheMachine(t *testing.T) {
	f := newT3Fixture(t)
	_, deploy := f.control(t, f.deploy)
	_, alice := f.control(t, f.alice)
	ctx := t.Context()
	changes := []SecretChange{{Slug: "api", Variables: []string{"DB_PASSWORD"}}}

	require.NoError(t, wsjson.Write(ctx, alice, Frame{Type: FrameSecretsChanged, SecretChanges: changes}))
	require.NoError(t, wsjson.Write(ctx, deploy, Frame{Type: FrameSecretsChanged, SecretChanges: changes}))

	eventually(t, 2*time.Second, func() bool { return len(f.bus.topicEvents(TopicSecretsChanged)) > 0 })
	require.Len(t, f.bus.topicEvents(TopicSecretsChanged), 1, "a personal runner's report is dropped")
	var ev SecretsChangedReport
	require.NoError(t, json.Unmarshal(f.bus.topicEvents(TopicSecretsChanged)[0].Payload, &ev))
	assert.Equal(t, SecretsChangedReport{Machine: "m1", Changes: changes}, ev, "the machine is the runner's own, never the frame's")
}
