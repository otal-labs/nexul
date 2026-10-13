// Package welcomestep is the vocabulary of Welcome wizard steps a role can give: roles store the keys, welcome reads them.
package welcomestep

import (
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// The steps a role can tick; Profile is not one, since every Welcome wizard opens on it.
const (
	GitHub       = "github"
	Computer     = "computer"
	Projects     = "projects"
	DeployServer = "deploy_server"
	Phone        = "phone"
)

// All lists every step in the order the wizard shows them.
var All = []string{GitHub, Computer, Projects, DeployServer, Phone}

// Owner is the Owner role's steps, fixed in code like the rest of the role.
var Owner = []string{GitHub, Computer}

// Parse checks every key is a step and returns them once each, in the wizard's order; nil stays nil.
func Parse(keys []string) ([]string, error) {
	if keys == nil {
		return nil, nil
	}
	for _, k := range keys {
		if !slices.Contains(All, k) {
			return nil, fmt.Errorf("%w: %q is not a welcome step; the steps are %s", apperrs.ErrInvalid, k, strings.Join(All, ", "))
		}
	}
	out := make([]string, 0, len(keys))
	for _, k := range All {
		if slices.Contains(keys, k) {
			out = append(out, k)
		}
	}
	return out, nil
}

// Added returns the steps in after that before lacks, which is what a save gives the role's holders again.
func Added(before, after []string) []string {
	var out []string
	for _, k := range after {
		if !slices.Contains(before, k) {
			out = append(out, k)
		}
	}
	return out
}
