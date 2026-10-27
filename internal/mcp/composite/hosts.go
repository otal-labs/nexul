package composite

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// HostKind is one kind of enrolled host (runner, automations host); the composition root adapts each domain to it.
type HostKind interface {
	// Enroll mints a one-time enrollment code for a host named name, returning the code and its install commands.
	Enroll(ctx context.Context, name, machine string) (any, error)
	// Remove revokes the host's credential, deletes it, and tells it to uninstall itself.
	Remove(ctx context.Context, id string) error
}

// CodeRevoker is a host kind whose unused enrollment codes can be revoked.
type CodeRevoker interface {
	RevokeCode(ctx context.Context, id string) error
}

// HostTools are the enroll and remove tools shared by every host kind, so a new kind costs no tool.
func HostTools(kinds map[string]HostKind) []mcptool.Tool {
	return []mcptool.Tool{hostCreateTool(kinds), hostDeleteTool(kinds)}
}

type hostCreateIn struct {
	Kind    string `json:"kind" jsonschema:"What to enroll: runner or automations."`
	Name    string `json:"name" jsonschema:"The new host's name, 1 to 32 lowercase letters, digits or dashes, e.g. build-box-1."`
	Machine string `json:"machine,omitempty" jsonschema:"The machine it runs on, by name from machine_list; omitted, it is filed under a machine named after it."`
}

type hostDeleteIn struct {
	Kind   string `json:"kind" jsonschema:"What to remove: runner, automations or machine."`
	ID     string `json:"id,omitempty" jsonschema:"The host's or machine's id, from machine_list. Send this or code_id."`
	CodeID string `json:"code_id,omitempty" jsonschema:"An unused enrollment code's id, from host_create's result or machine_list's open_enrollments, to revoke instead of removing a host. Only a runner's codes can be revoked."`
}

func hostCreateTool(kinds map[string]HostKind) mcptool.Tool {
	return mcptool.New("host_create", "Enroll host",
		"Enrolls a new runner or automations host and returns a one-time enrollment code, shown only in this result "+
			"and valid for one hour, with the install commands that carry it for Linux or macOS and for Windows. Run one "+
			"on the machine. The result's id revokes an unused runner code with host_delete. A runner needs Manage on an "+
			"existing machine or Add machines for a new one; an automations host needs Manage on an existing machine, and "+
			"with no machine or an unknown one is the instance Owner's alone.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in hostCreateIn) (any, error) {
			kind, err := hostKind(kinds, in.Kind)
			if err != nil {
				return nil, err
			}
			return kind.Enroll(ctx, in.Name, in.Machine)
		})
}

type hostDeleted struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

func hostDeleteTool(kinds map[string]HostKind) mcptool.Tool {
	return mcptool.New("host_delete", "Remove host",
		"Removes a runner or automations host, deletes a machine, or with code_id revokes an unused runner enrollment "+
			"code. A removed host can never connect again, and a connected one uninstalls itself (an offline one when it "+
			"returns). A host needs Manage on its machine. Deleting a machine (kind machine) needs Delete on it and is "+
			"refused, naming each, while a stack or gateway targets it or a runner on it is connected; its offline "+
			"runners and rules go with it. A code is revoked by its maker, a manager of its machine, or the instance "+
			"Owner.",
		mcptool.Hints{},
		func(ctx context.Context, in hostDeleteIn) (any, error) {
			kind, err := hostKind(kinds, in.Kind)
			if err != nil {
				return nil, err
			}
			if (in.ID == "") == (in.CodeID == "") {
				return nil, fmt.Errorf("%w: send either id to remove a host or code_id to revoke an unused code", apperrs.ErrInvalid)
			}
			if in.CodeID != "" {
				return revokeCode(ctx, kind, in)
			}
			err = kind.Remove(ctx, in.ID)
			if errors.Is(err, apperrs.ErrNotFound) {
				return nil, fmt.Errorf("%w; machine_list lists hosts with their ids", err)
			}
			if err != nil {
				return nil, err
			}
			return hostDeleted{ID: in.ID, Deleted: true}, nil
		})
}

func revokeCode(ctx context.Context, kind HostKind, in hostDeleteIn) (any, error) {
	revoker, ok := kind.(CodeRevoker)
	if !ok {
		return nil, fmt.Errorf("%w: kind %q has no enrollment codes to revoke here; only runner codes can be", apperrs.ErrInvalid, in.Kind)
	}
	err := revoker.RevokeCode(ctx, in.CodeID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("%w; machine_list lists the unused codes you may revoke", err)
	}
	if err != nil {
		return nil, err
	}
	return hostDeleted{ID: in.CodeID, Deleted: true}, nil
}

func hostKind(kinds map[string]HostKind, name string) (HostKind, error) {
	if kind, ok := kinds[name]; ok {
		return kind, nil
	}
	known := make([]string, 0, len(kinds))
	for k := range kinds {
		known = append(known, k)
	}
	slices.Sort(known)
	return nil, fmt.Errorf("%w: kind %q cannot be enrolled or removed here; use one of: %s", apperrs.ErrInvalid, name, strings.Join(known, ", "))
}
