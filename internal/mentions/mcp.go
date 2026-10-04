package mentions

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// mentionSearchScan is the most hits Search returns; the picker never needs more.
const mentionSearchScan = 50

type mentionRef struct {
	Type string `json:"type" jsonschema:"What the reference points at: ticket, doc, or person."`
	ID   string `json:"id" jsonschema:"The ticket's id or key such as REF-102, the doc's id, or a person's user id."`
}

type mentionSearchIn struct {
	Query       string       `json:"query,omitempty" jsonschema:"Text to match against ticket and doc titles and bodies, ticket notes, and people's logins and display names, or a ticket key such as REF-102, which sorts first."`
	WorkspaceID string       `json:"workspace_id,omitempty" jsonschema:"Only find people, and the ticket a key such as REF-102 names, in this workspace; omit to search every workspace you share. With refs, a ticket ref given as a key resolves in this workspace."`
	Refs        []mentionRef `json:"refs,omitempty" jsonschema:"References to resolve to live chips instead of searching, for example the mentions found in a document."`
	mcptool.PageArgs
}

// MCPTools returns the mention tool; agents reference targets with the same markdown links people do.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("mention_search", "Search mentions",
			"Finds people, tickets, and docs to @-mention, or with refs resolves references already in a document to live chips. "+
				"Send exactly one of query or refs. A search returns at most 50 matches, each with its type, id, title, and status; "+
				"a person's title is their display name and carries their login. "+
				"refs return the current title, status, project key parts, and whether you can open it, and leave out targets that no longer exist. "+
				"Docs you cannot open are never found by a search. To mention a person in a doc or ticket body, write [@login](/people/<id>). "+
				"Use doc_list or ticket_list for filtered browsing.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in mentionSearchIn) (any, error) {
				if (in.Query == "") == (len(in.Refs) == 0) {
					return nil, fmt.Errorf("%w: send exactly one of query or refs", apperrs.ErrInvalid)
				}
				if in.Query != "" {
					results, err := s.Search(ctx, in.Query, in.WorkspaceID, mentionSearchScan)
					if err != nil {
						return nil, err
					}
					return mcptool.Paginate(results, in.PageArgs), nil
				}
				refs, err := toRefs(in.Refs)
				if err != nil {
					return nil, err
				}
				chips, err := s.Resolve(ctx, in.WorkspaceID, refs)
				if err != nil {
					return nil, err
				}
				return mcptool.Paginate(chips, in.PageArgs), nil
			}),
	}
}

func toRefs(in []mentionRef) ([]Ref, error) {
	out := make([]Ref, 0, len(in))
	for _, r := range in {
		if r.Type != string(KindTicket) && r.Type != string(KindDoc) && r.Type != string(KindPerson) {
			return nil, fmt.Errorf("%w: ref type %q must be ticket, doc, or person", apperrs.ErrInvalid, r.Type)
		}
		out = append(out, Ref(r))
	}
	return out, nil
}
