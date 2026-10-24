package crashintake

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReport_RefusesWhatIsNotACrashReport(t *testing.T) {
	without := func(name string) map[string]string {
		f := reportFields()
		delete(f, name)
		return f
	}
	with := func(name, value string) map[string]string {
		f := reportFields()
		f[name] = value
		return f
	}
	tests := []struct {
		name string
		post Post
		want string
	}{
		{"plain message", Post{Content: "deploy finished"}, "not a crash fingerprint line"},
		{"short fingerprint", Post{Content: FingerprintLine + "abc"}, "not a crash fingerprint line"},
		{"no embed", Post{Content: FingerprintLine + fp}, "no embed"},
		{"embeds not a list", Post{Content: FingerprintLine + fp, Embeds: json.RawMessage(`{}`)}, "not a list"},
		{"another contract", postOf(with("Contract", "crash-report/2")), `"crash-report/2"`},
		{"fingerprints differ", postOf(with("Fingerprint", strings.Repeat("0", 64))), "differs"},
		{"unknown kind", postOf(with("Kind", "panic")), `kind "panic"`},
		{"no app version", postOf(without("App version")), "app version"},
		{"short build commit", postOf(with("Build commit", "abc123")), "full commit id"},
		{"bad suppressed count", postOf(with("Suppressed duplicates", "-1")), "not a count"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseReport(tt.post.Content, tt.post.Embeds)
			require.ErrorIs(t, err, errNotAReport)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestParseReport_ReadsEveryField(t *testing.T) {
	f := reportFields()
	f["Build commit"], f["Truncated"], f["Suppressed duplicates"] = strings.Repeat("d", 64), "true", "2"
	p := postOf(f)

	r, err := ParseReport(" "+p.Content+"\n", p.Embeds)
	require.NoError(t, err)

	assert.Equal(t, Report{
		Fingerprint: fp, Kind: "ui-error", ProcessType: "renderer", AppVersion: "1.4.0", BuildCommit: strings.Repeat("d", 64),
		OS: "linux", Arch: "x64", FirstSeen: "2026-10-20T10:00:00Z", Signature: "TypeError", Stack: "renderPanel\nloadPanel",
		Truncated: true, Suppressed: 2,
	}, r)
	assert.Equal(t, 3, r.Reports())
}

func TestTopFrame_SkipsWhatTheSenderRedacted(t *testing.T) {
	assert.Equal(t, "loadPanel", Report{Stack: "[redacted]\nloadPanel"}.TopFrame())
	assert.Empty(t, Report{Stack: "none"}.TopFrame())
}

func TestBugTitle_WithoutANamedFrame(t *testing.T) {
	assert.Equal(t, "Crash: SIGSEGV (native-process)", bugTitle(Report{Signature: "SIGSEGV", Kind: "native-process", Stack: "none"}))
	assert.Equal(t, "Crash: [redacted] (ui-error)", bugTitle(Report{Kind: "ui-error"}))
}

func TestBugBody_FencesAStackThatHoldsBackticks(t *testing.T) {
	body := bugBody(Report{Stack: "a```b", Truncated: true}, false)
	assert.Contains(t, body, "````text\na```b\n````")
	assert.Contains(t, body, "cut this report to its size cap")
}

func TestParseTally_AnEditedCountStartsOver(t *testing.T) {
	assert.Equal(t, tally{versions: []string{"1.0"}}, parseTally("Reports: many\nVersions: 1.0,"))
}
