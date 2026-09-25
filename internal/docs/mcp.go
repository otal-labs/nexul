package docs

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// ponytail: search ranks at most 1,000 hits and pages them in memory; add FTS offsets if a query ever matches more.
const docSearchScan = 1000

// docResult is a doc as the model reads it: the body as markdown, never the stored rich-text tree.
type docResult struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Version   int       `json:"version"`
	Archived  bool      `json:"archived"`
	UpdatedAt time.Time `json:"updated_at"`
}

type docListIn struct {
	Query           string `json:"query,omitempty" jsonschema:"Full-text search over titles and bodies, for example deploy rollback. Omit to browse. Archived docs never match a search."`
	ProjectID       string `json:"project_id,omitempty" jsonschema:"Only docs in this project; project_list lists projects. Omit for every project."`
	IncludeArchived bool   `json:"include_archived,omitzero" jsonschema:"Also list archived docs when browsing without a query. Defaults to false."`
	mcptool.PageArgs
}

type docGetIn struct {
	ID string `json:"id" jsonschema:"The doc's id, from doc_list."`
}

type docCreateIn struct {
	ProjectID string `json:"project_id" jsonschema:"The project the doc belongs to; project_list lists projects."`
	Title     string `json:"title" jsonschema:"The doc's title, for example Storage spine."`
	Body      string `json:"body,omitempty" jsonschema:"The doc's body as markdown. Omit for an empty doc."`
}

type docUpdateIn struct {
	ID       string  `json:"id" jsonschema:"The doc's id, from doc_list."`
	Title    *string `json:"title,omitempty" jsonschema:"New title. Omit to keep the current one."`
	Body     *string `json:"body,omitempty" jsonschema:"New body as markdown, replacing the whole body. Omit to keep the current body."`
	Archived *bool   `json:"archived,omitempty" jsonschema:"true archives the doc (hidden from search), false restores it. Omit to leave it as is."`
}

// MCPTools returns the docs tools.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{docListTool(s), docGetTool(s), docCreateTool(s), docUpdateTool(s)}
}

func docListTool(s *Service) mcptool.Tool {
	return mcptool.New("doc_list", "List docs",
		"Lists docs by title, or full-text searches them when query is set, optionally within one project. "+
			"Use it to find a doc's id, then doc_get to read its body. "+
			"Returns items without bodies, ranked by relevance with a query and oldest first without one; "+
			"a doc you cannot open is listed with can_open false when browsing and left out of a search. "+
			"Archived docs are hidden unless include_archived is set.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in docListIn) (any, error) {
			items, err := listDocs(ctx, s, in.ProjectID)
			if err != nil {
				return nil, err
			}
			if in.Query != "" {
				if items, err = rankBySearch(ctx, s, in.Query, items); err != nil {
					return nil, err
				}
			}
			if !in.IncludeArchived {
				items = slices.DeleteFunc(items, func(d *DocListItem) bool { return d.Archived })
			}
			return mcptool.Paginate(items, in.PageArgs), nil
		})
}

func listDocs(ctx context.Context, s *Service, projectID string) ([]*DocListItem, error) {
	if projectID != "" {
		return s.ListByProject(ctx, projectID)
	}
	return s.List(ctx)
}

// rankBySearch keeps the listed docs the search matched, in the search's rank order.
func rankBySearch(ctx context.Context, s *Service, query string, items []*DocListItem) ([]*DocListItem, error) {
	hits, err := s.Search(ctx, query, docSearchScan)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*DocListItem, len(items))
	for _, d := range items {
		byID[d.ID] = d
	}
	ranked := make([]*DocListItem, 0, len(hits))
	for _, h := range hits {
		if d, ok := byID[h.ID]; ok {
			ranked = append(ranked, d)
		}
	}
	return ranked, nil
}

func docGetTool(s *Service) mcptool.Tool {
	return mcptool.New("doc_get", "Get doc",
		"Returns one doc with its full body as markdown, its project, version, and archived state. "+
			"Use it after doc_list has given you the id, and before doc_update so you edit the current text. "+
			"Fails with forbidden when you cannot read the doc.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in docGetIn) (any, error) {
			d, err := s.Get(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			return toDocResult(d)
		})
}

func docCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("doc_create", "Create doc",
		"Creates a doc in a project from a markdown title and body, and makes you its owner. "+
			"Use it for documentation and requirements people read; notes meant for agents belong in memory_create instead. "+
			"Returns the new doc with its body as markdown.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in docCreateIn) (any, error) {
			d, err := s.Create(ctx, in.ProjectID, in.Title, in.Body)
			if err != nil {
				return nil, err
			}
			return toDocResult(d)
		})
}

func docUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("doc_update", "Update doc",
		"Changes a doc's title, body, or archived state; only the fields you send change. "+
			"A new title or body saves a new version, and archived true hides the doc from search until archived false restores it. "+
			"Read the doc with doc_get first, because body replaces the whole body. "+
			"Returns the doc as it now stands.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in docUpdateIn) (any, error) {
			d, err := s.Get(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			saved := in.Title != nil || in.Body != nil
			if saved {
				if d, err = s.Update(ctx, in.ID, deref(in.Title, d.Title), deref(in.Body, d.Body)); err != nil {
					return nil, err
				}
			}
			if in.Archived != nil && *in.Archived != d.Archived {
				if d, err = setArchived(ctx, s, in.ID, *in.Archived); err != nil {
					return nil, archiveErr(saved, err)
				}
			}
			return toDocResult(d)
		})
}

func setArchived(ctx context.Context, s *Service, id string, archived bool) (*Doc, error) {
	if archived {
		return s.Archive(ctx, id)
	}
	return s.Restore(ctx, id)
}

// archiveErr says the title and body were already saved when only the archive step failed.
func archiveErr(saved bool, err error) error {
	if saved {
		return fmt.Errorf("the title and body were saved, but changing archived failed: %w", err)
	}
	return err
}

func deref[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

func toDocResult(d *Doc) (docResult, error) {
	md, err := richtext.ToMarkdown(d.Body)
	if err != nil {
		return docResult{}, fmt.Errorf("render doc %s: %w", d.ID, err)
	}
	return docResult{
		ID: d.ID, ProjectID: d.ProjectID, Title: d.Title, Body: md,
		Version: d.Version, Archived: d.Archived, UpdatedAt: d.UpdatedAt,
	}, nil
}
