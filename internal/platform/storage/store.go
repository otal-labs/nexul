package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"modernc.org/sqlite"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

type Store struct {
	db                    *sql.DB
	w                     *Serializer
	Docs                  *DocsRepo
	Memories              *MemoriesRepo
	Attachments           *AttachmentsRepo
	Tickets               *TicketsRepo
	CodeReviews           *CodeReviewsRepo
	Deploys               *DeploysRepo
	Stacks                *StacksRepo
	Services              *ServicesRepo
	Topology              *TopologyRepo
	Runners               *RunnersRepo
	Machines              *MachinesRepo
	InstanceUpgrades      *InstanceUpgradesRepo
	Outbox                *OutboxRepo
	ProcessedEvents       *ProcessedEventsRepo
	DeadLetters           *DeadLettersRepo
	Users                 *UsersRepo
	Allowlist             *AllowlistRepo
	Settings              *SettingsRepo
	Access                *AccessRepo
	PATs                  *PATsRepo
	Projects              *ProjectsRepo
	Categories            *CategoriesRepo
	TicketTypes           *TicketTypesRepo
	Statuses              *StatusesRepo
	Workspaces            *WorkspacesRepo
	WorkspaceMembers      *WorkspaceMembersRepo
	WorkspaceInvites      *WorkspaceInvitesRepo
	Invitations           *InvitationsRepo
	OAuthHandoffs         *OAuthHandoffsRepo
	Roles                 *RolesRepo
	Plays                 *PlaysRepo
	PlayTrails            *PlayTrailsRepo
	DNS                   *DNSRepo
	Automations           *AutomationsRepo
	AutomationVersions    *AutomationVersionsRepo
	AutomationSecrets     *AutomationSecretsRepo
	AutomationRuns        *AutomationRunsRepo
	AutomationCursors     *AutomationCursorsRepo
	AutomationEventLog    *AutomationEventLogRepo
	Notifications         *NotificationsRepo
	Collab                *CollabRepo
	IntegrationInstalls   *IntegrationInstallsRepo
	IntegrationTokens     *IntegrationTokensRepo
	IntegrationSubs       *IntegrationSubscriptionsRepo
	IntegrationDeliveries *IntegrationDeliveriesRepo
	EventSchemas          *EventSchemasRepo
	Audit                 *AuditRepo
	Connectors            *ConnectorsRepo
	ConnectorAppConfig    *ConnectorAppConfigRepo
	Chat                  *ChatRepo
	Pairing               *PairingRepo
}

// New builds a Store over db; encKey is the AES-256 key repos use to encrypt secrets, derived once at startup.
func New(db *sql.DB, encKey []byte) *Store {
	w := &Serializer{}
	q := sqlcgen.New(db)
	return &Store{
		db:                    db,
		w:                     w,
		Docs:                  &DocsRepo{db: db, w: w, q: q},
		Memories:              &MemoriesRepo{db: db, w: w, q: q},
		Attachments:           &AttachmentsRepo{db: db, w: w, q: q},
		Tickets:               &TicketsRepo{db: db, w: w, q: q},
		CodeReviews:           &CodeReviewsRepo{db: db, w: w, q: q},
		Deploys:               &DeploysRepo{db: db, w: w, q: q},
		Stacks:                &StacksRepo{db: db, w: w, q: q},
		Services:              &ServicesRepo{db: db, w: w, q: q},
		Topology:              &TopologyRepo{db: db, w: w, q: q},
		Runners:               &RunnersRepo{db: db, w: w, q: q},
		Machines:              &MachinesRepo{db: db, w: w, q: q},
		InstanceUpgrades:      &InstanceUpgradesRepo{db: db, w: w, q: q},
		Outbox:                &OutboxRepo{db: db, w: w, q: q},
		ProcessedEvents:       &ProcessedEventsRepo{db: db, w: w, q: q},
		DeadLetters:           &DeadLettersRepo{db: db, w: w, q: q},
		Users:                 &UsersRepo{db: db, w: w, q: q},
		Allowlist:             &AllowlistRepo{db: db, w: w, q: q},
		Settings:              &SettingsRepo{db: db, w: w, q: q, encKey: encKey},
		Access:                &AccessRepo{db: db, w: w, q: q},
		PATs:                  &PATsRepo{db: db, w: w, q: q},
		Projects:              &ProjectsRepo{db: db, w: w, q: q},
		Categories:            &CategoriesRepo{db: db, w: w, q: q},
		TicketTypes:           &TicketTypesRepo{db: db, w: w, q: q},
		Statuses:              &StatusesRepo{db: db, w: w, q: q},
		Workspaces:            &WorkspacesRepo{db: db, w: w, q: q},
		WorkspaceMembers:      &WorkspaceMembersRepo{db: db, w: w, q: q},
		WorkspaceInvites:      &WorkspaceInvitesRepo{db: db, w: w, q: q},
		Invitations:           &InvitationsRepo{db: db, w: w, q: q},
		OAuthHandoffs:         &OAuthHandoffsRepo{db: db, w: w, q: q},
		Roles:                 &RolesRepo{db: db, w: w, q: q},
		Plays:                 &PlaysRepo{db: db, w: w, q: q},
		PlayTrails:            &PlayTrailsRepo{db: db, w: w, q: q},
		DNS:                   &DNSRepo{db: db, w: w, q: q},
		Automations:           &AutomationsRepo{db: db, w: w, q: q},
		AutomationVersions:    &AutomationVersionsRepo{db: db, w: w, q: q},
		AutomationSecrets:     &AutomationSecretsRepo{db: db, w: w, q: q, encKey: encKey},
		AutomationRuns:        &AutomationRunsRepo{db: db, w: w, q: q},
		AutomationCursors:     &AutomationCursorsRepo{db: db, w: w, q: q},
		AutomationEventLog:    &AutomationEventLogRepo{db: db, q: q},
		Notifications:         &NotificationsRepo{db: db, w: w, q: q},
		Collab:                &CollabRepo{db: db, w: w, q: q},
		IntegrationInstalls:   &IntegrationInstallsRepo{db: db, w: w, q: q},
		IntegrationTokens:     &IntegrationTokensRepo{db: db, w: w, q: q},
		IntegrationSubs:       &IntegrationSubscriptionsRepo{db: db, w: w, q: q},
		IntegrationDeliveries: &IntegrationDeliveriesRepo{db: db, w: w, q: q},
		EventSchemas:          &EventSchemasRepo{db: db, w: w, q: q},
		Audit:                 &AuditRepo{db: db, w: w, q: q},
		Connectors:            &ConnectorsRepo{db: db, w: w, q: q, encKey: encKey},
		ConnectorAppConfig:    &ConnectorAppConfigRepo{db: db, w: w, q: q, encKey: encKey},
		Chat:                  &ChatRepo{db: db, w: w, q: q},
		Pairing:               &PairingRepo{db: db, w: w, q: q},
	}
}

func (s *Store) Close() error {
	return s.db.Close()
}

const sqliteConstraint = 19

func classifyWriteErr(err error) error {
	var se *sqlite.Error
	if errors.As(err, &se) && se.Code()&0xff == sqliteConstraint {
		return apperrs.ErrConflict
	}
	return err
}

// notFoundIfNoRows maps the driver's empty-result error onto the domain sentinel every repo returns.
func notFoundIfNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return apperrs.ErrNotFound
	}
	return err
}

// optionalInt reads a nullable integer aggregate; sqlc types MAX(...) as any because an empty set yields NULL.
func optionalInt(v any) (int64, bool, error) {
	if v == nil {
		return 0, false, nil
	}
	n, ok := v.(int64)
	if !ok {
		return 0, false, fmt.Errorf("unexpected aggregate type %T", v)
	}
	return n, true, nil
}

// nullString wraps a required lookup key for a nullable column; sqlc types those params as sql.NullString.
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}
