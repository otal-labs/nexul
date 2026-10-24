// Package crashintake turns crash reports a bot posts into bug tickets: one per crash, a running note for its repeats,
// and a new high-priority bug when a crash returns in a build that holds its fix.
package crashintake

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Contract is the only report format the intake reads; any other post is ignored.
const Contract = "crash-report/1"

// FingerprintLine is the fixed body line a crash's tickets carry, which the intake searches for.
const FingerprintLine = "Crash fingerprint: "

var (
	errNotAReport = errors.New("not a crash report")

	fingerprintPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	commitPattern      = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)
	kinds              = []string{"native-process", "native-fatal", "ui-error"}
)

// Report is one crash-report/1 post: the fingerprint that identifies the crash and what the sending build said about it.
type Report struct {
	Fingerprint string
	Kind        string
	ProcessType string
	AppVersion  string
	BuildCommit string
	OS          string
	Arch        string
	FirstSeen   string
	Signature   string
	// Stack is the sender's already redacted frames, one per line, or "none".
	Stack     string
	Truncated bool
	// Suppressed is how many repeats the sender held back since its last accepted report.
	Suppressed int
}

// Reports is this post's count toward a running tally: itself plus the repeats its sender held back.
func (r Report) Reports() int { return 1 + r.Suppressed }

// TopFrame is the newest stack frame the sender could name, or "" when it named none.
func (r Report) TopFrame() string {
	for line := range strings.SplitSeq(r.Stack, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && line != "none" && line != "[redacted]" {
			return line
		}
	}
	return ""
}

type embed struct {
	Fields []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"fields"`
}

// ParseReport reads a bot post as a crash-report/1 report; a post that is anything else is an errNotAReport error naming why.
func ParseReport(content string, embeds json.RawMessage) (Report, error) {
	fingerprint, ok := strings.CutPrefix(strings.TrimSpace(content), FingerprintLine)
	if !ok || !fingerprintPattern.MatchString(fingerprint) {
		return Report{}, fmt.Errorf("%w: the content is not a crash fingerprint line", errNotAReport)
	}
	var es []embed
	if len(embeds) > 0 {
		if err := json.Unmarshal(embeds, &es); err != nil {
			return Report{}, fmt.Errorf("%w: the embeds are not a list: %w", errNotAReport, err)
		}
	}
	if len(es) == 0 {
		return Report{}, fmt.Errorf("%w: the post has no embed", errNotAReport)
	}
	fields := map[string]string{}
	for _, f := range es[0].Fields {
		if _, seen := fields[f.Name]; !seen {
			fields[f.Name] = strings.TrimSpace(f.Value)
		}
	}
	if fields["Contract"] != Contract {
		return Report{}, fmt.Errorf("%w: the contract is %q, not %s", errNotAReport, fields["Contract"], Contract)
	}
	if fields["Fingerprint"] != fingerprint {
		return Report{}, fmt.Errorf("%w: the embed's fingerprint differs from the content's", errNotAReport)
	}
	return reportFrom(fingerprint, fields)
}

func reportFrom(fingerprint string, fields map[string]string) (Report, error) {
	r := Report{
		Fingerprint: fingerprint,
		Kind:        fields["Kind"],
		ProcessType: fields["Process type"],
		AppVersion:  fields["App version"],
		BuildCommit: fields["Build commit"],
		OS:          fields["OS"],
		Arch:        fields["Arch"],
		FirstSeen:   fields["First seen"],
		Signature:   fields["Signature"],
		Stack:       fields["Stack"],
		Truncated:   fields["Truncated"] == "true",
	}
	if !slices.Contains(kinds, r.Kind) {
		return Report{}, fmt.Errorf("%w: kind %q is not one of %s", errNotAReport, r.Kind, strings.Join(kinds, ", "))
	}
	if r.AppVersion == "" {
		return Report{}, fmt.Errorf("%w: the app version is missing", errNotAReport)
	}
	if !commitPattern.MatchString(r.BuildCommit) {
		return Report{}, fmt.Errorf("%w: the build commit is not a full commit id", errNotAReport)
	}
	suppressed, ok := fields["Suppressed duplicates"]
	if !ok {
		return r, nil
	}
	n, err := strconv.Atoi(suppressed)
	if err != nil || n < 0 {
		return Report{}, fmt.Errorf("%w: suppressed duplicates %q is not a count", errNotAReport, suppressed)
	}
	r.Suppressed = n
	return r, nil
}
