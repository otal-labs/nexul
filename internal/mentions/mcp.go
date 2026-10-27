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
	WorkspaceID string       `json:"workspace_id,omitempty" jsonschema:"Only find people, tickets, and docs in this workspace, the ticket a key such as REF-102 names included; omit to search every workspace you share. Required with kind. With refs, a ticket ref given as a key resolves in this workspace."`
	Kind        string       `json:"kind,omitempty" jsonschema:"Search like the app's search box instead of for mention targets: all for every kind, or one of ticket, doc, pull_request, stack, automation, channel, memory for more of that kind. Omit to find people, tickets, and docs to mention."`
	Refs        []mentionRef `json:"refs,omitempty" jsonschema:"References to resolve to live chips instead of searching."`
	mcptool.PageArgs
}

// findPerKind is how many title hits and body hits each kind returns when kind is all.
const findPerKind = 10

// foundPage is a page of search box hits plus the kinds that had more than findPerKind.
type foundPage struct {
	mcptool.Page[Hit]
	More []string `json:"more"`
}

// MCPTools returns the mention tool; agents reference targets with the same markdown links people do.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("mention_search", "Search mentions",
			"Finds people, tickets, and docs to @-mention, or with refs resolves references already in a document to live chips. "+
				"A search returns at most 50 matches, each with its type, id, title, and status; "+
				"a person's title is their display name and carries their login. "+
				"refs return the current title, status, project key parts, and whether you can open it, and leave out targets that no longer exist. "+
				"With kind it searches like the app's search box over one workspace: tickets, docs, pull requests (which open their ticket), stacks, automations, channels, and memories, "+
				"each hit marked in_body when only its body or notes matched; kind all returns up to 10 title and 10 body hits per kind and names the kinds cut short in more, "+
				"and a single kind returns up to 50 of each. "+
				"To mention a person in a doc or ticket body, write [@login](/people/<id>). "+
				"Use doc_list or ticket_list for filtered browsing.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in mentionSearchIn) (any, error) {
				if (in.Query == "") == (len(in.Refs) == 0) {
					return nil, fmt.Errorf("%w: send exactly one of query or refs", apperrs.ErrInvalid)
				}
				if in.Kind != "" {
					return find(ctx, s, in)
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

func find(ctx context.Context, s *Service, in mentionSearchIn) (any, error) {
	if in.Query == "" {
		return nil, fmt.Errorf("%w: kind needs a query", apperrs.ErrInvalid)
	}
	kind, limit := in.Kind, 50
	if kind == "all" {
		kind, limit = "", findPerKind
	}
	found, err := s.Find(ctx, in.Query, in.WorkspaceID, kind, limit)
	if err != nil {
		return nil, err
	}
	return foundPage{Page: mcptool.Paginate(found.Hits, in.PageArgs), More: found.More}, nil
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
