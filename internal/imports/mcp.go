package imports

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the one import tool every source shares (ADR 0201).
func MCPTools(svc *Service) []mcptool.Tool {
	return []mcptool.Tool{importRunTool(svc)}
}

type importRunIn struct {
	ID         string `json:"id,omitempty" jsonschema:"An import's id: alone it reads progress; with confirm or cancel it acts."`
	Source     string `json:"source,omitempty" jsonschema:"Where to import from: github, one attached repository's issues; linear, one team's; trello, a board; asana, a project; notion, the pages shared with an integration."`
	ProjectID  string `json:"project_id,omitempty" jsonschema:"The project to import into, from project_list."`
	From       string `json:"from,omitempty" jsonschema:"What to import, omitted when there is one: github owner/name; linear a team key; trello a board, asana a project, by URL or id; notion page links, omitted for all top pages."`
	Credential string `json:"credential,omitempty" jsonschema:"The source's token: Linear personal API key, Trello key:token, Asana personal access token, Notion integration secret; github none. Sealed, deleted when the import ends, never returned."`
	Confirm    bool   `json:"confirm,omitempty" jsonschema:"With id: write a scanned import that is ready."`
	Cancel     bool   `json:"cancel,omitempty" jsonschema:"With id: stop the import; what it already wrote stays."`
}

func importRunTool(svc *Service) mcptool.Tool {
	return mcptool.New("import_run", "Run import",
		"Imports another tool's records into a project: issues as tickets with their comments, labels, assignee, status "+
			"and files, pages as docs; only the workspace's Owner may. With source and project_id it starts a scan that counts what would come "+
			"in and how people and statuses map, and returns the import; with that id it returns progress, and confirm "+
			"writes it once its state is ready, within 24 hours. A re-run skips what an earlier import brought in, and "+
			"nothing imported notifies anyone or starts an auto play.",
		mcptool.Hints{},
		func(ctx context.Context, in importRunIn) (any, error) {
			if in.ID == "" {
				if in.Confirm || in.Cancel {
					return nil, fmt.Errorf("%w: confirm and cancel need the import's id", apperrs.ErrInvalid)
				}
				return svc.Start(ctx, Start{Source: in.Source, ProjectID: in.ProjectID, From: in.From, Credential: in.Credential})
			}
			if in.Confirm && in.Cancel {
				return nil, fmt.Errorf("%w: give confirm or cancel, not both", apperrs.ErrInvalid)
			}
			if in.Confirm {
				return svc.Confirm(ctx, in.ID)
			}
			if in.Cancel {
				return svc.Cancel(ctx, in.ID)
			}
			return svc.Get(ctx, in.ID)
		})
}
