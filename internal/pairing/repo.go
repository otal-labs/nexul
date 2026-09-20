package pairing

import "context"

// Repo is the slice of SQLite the pairing domain needs; bearer tokens arrive already encrypted.
type Repo interface {
	// SaveComputer upserts a computer row (create on first pairing, update in place on re-pair).
	SaveComputer(ctx context.Context, c Computer) error
	// GetComputer returns one of userID's own computers, or ErrNotFound, never leaking a mismatched owner's row.
	GetComputer(ctx context.Context, userID, id string) (*Computer, error)
	// ListComputers returns userID's paired computers, newest first.
	ListComputers(ctx context.Context, userID string) ([]Computer, error)
	// DeleteComputer removes one of userID's own computers, or ErrNotFound.
	DeleteComputer(ctx context.Context, userID, id string) error

	// GetDefaults returns userID's defaults, or a zero Defaults if never set (not an error, defaults are optional).
	GetDefaults(ctx context.Context, userID string) (Defaults, error)
	// SaveDefaults upserts userID's defaults row.
	SaveDefaults(ctx context.Context, d Defaults) error

	// GetComputerByID returns a computer by id regardless of owner; ResolveTarget uses it for project-linked computers.
	GetComputerByID(ctx context.Context, id string) (*Computer, error)

	// GetProjectLink returns projectID's link, or a zero ProjectLink if never linked (optional, like GetDefaults).
	GetProjectLink(ctx context.Context, projectID string) (ProjectLink, error)
	// SaveProjectLink upserts a project's link row.
	SaveProjectLink(ctx context.Context, link ProjectLink) error
	// DeleteProjectLink clears projectID's link. A no-op if the project was never linked.
	DeleteProjectLink(ctx context.Context, projectID string) error
}
