// Package permissions defines the one domain x action vocabulary shared by roles, tokens, and the agent:
// read, write, and delete on every domain, plus a further verb a domain may declare (ADR 0057).
package permissions

import (
	"encoding/json"
	"slices"
	"strings"
)

// Action is one permission, "<domain>:<action>", action either read/write/delete or a domain-declared
// verb (ADR 0057); the same value gates a role, a token, and the agent.
type Action string

// Member is the empty action a permission check reads as "belongs to the workspace", for what every member may read.
const Member Action = ""

// Actions referenced by name from Go code; every other grid value is only ever built from the catalog.
const (
	DocsRead           Action = "docs:read"
	DocsWrite          Action = "docs:write"
	DocsDelete         Action = "docs:delete"
	DocsThread         Action = "docs:thread"
	PermissionsWrite   Action = "permissions:write"
	ProjectsRead       Action = "projects:read"
	ProjectsWrite      Action = "projects:write"
	ProjectsDelete     Action = "projects:delete"
	TicketsRead        Action = "tickets:read"
	TicketsWrite       Action = "tickets:write"
	TicketsDelete      Action = "tickets:delete"
	StacksRead         Action = "stacks:read"
	StacksWrite        Action = "stacks:write"
	StacksDelete       Action = "stacks:delete"
	DeploysRead        Action = "deploys:read"
	DeploysWrite       Action = "deploys:write"
	TopologyRead       Action = "topology:read"
	TopologyWrite      Action = "topology:write"
	TopologyDelete     Action = "topology:delete"
	RunnersRead        Action = "runners:read"
	RunnersWrite       Action = "runners:write"
	RunnersDelete      Action = "runners:delete"
	MachinesRead       Action = "machines:read"
	MachinesWrite      Action = "machines:write"
	DNSRead            Action = "dns:read"
	DNSWrite           Action = "dns:write"
	DNSDelete          Action = "dns:delete"
	ReviewsRead        Action = "reviews:read"
	ReposRead          Action = "repos:read"
	ConnectorsRead     Action = "connectors:read"
	ConnectorsWrite    Action = "connectors:write"
	ChatWrite          Action = "chat:write"
	WorkspacesWrite    Action = "workspaces:write"
	WorkspacesCreate   Action = "workspaces:create"
	MembersWrite       Action = "members:write"
	RolesWrite         Action = "roles:write"
	RolesClone         Action = "roles:clone"
	AutomationsRead    Action = "automations:read"
	AutomationsWrite   Action = "automations:write"
	AutomationsDelete  Action = "automations:delete"
	PlaysRead          Action = "plays:read"
	PlaysWrite         Action = "plays:write"
	PlaysDelete        Action = "plays:delete"
	PlaysRun           Action = "plays:run"
	MemoriesRead       Action = "memories:read"
	MemoriesWrite      Action = "memories:write"
	MemoriesDelete     Action = "memories:delete"
	MemoriesClone      Action = "memories:clone"
	InstanceRead       Action = "instance:read"
	InstanceWrite      Action = "instance:write"
	AccountsRead       Action = "accounts:read"
	AccountsWrite      Action = "accounts:write"
	AccountsDelete     Action = "accounts:delete"
	IntegrationsRead   Action = "integrations:read"
	IntegrationsWrite  Action = "integrations:write"
	IntegrationsDelete Action = "integrations:delete"
	AuditRead          Action = "audit:read"
)

const (
	read   = "read"
	write  = "write"
	delete = "delete"
	run    = "run"
	clone  = "clone"
	thread = "thread"
	create = "create"
)

// verbLabel is the owner-facing text for a domain-declared verb (ADR 0057): one line per verb, no per-domain switch.
var verbLabel = map[Action]string{
	PlaysRun:      "Run plays",
	MemoriesClone: "Clone memories to another project or workspace",
	RolesClone:    "Clone roles to another workspace",
	DocsThread:    "See doc threads",
	// The creator owns what they create, and an Owner holds every permission, so this one is as strong as Owner.
	WorkspacesCreate: "Create workspaces",
}

// domainInfo is one row of the grid: an API domain, its display name, and the actions its routes expose.
type domainInfo struct {
	domain  string
	display string
	actions []string
}

// domainTable is the single source every catalog, valid-action set, and web grid derives from (display order).
var domainTable = []domainInfo{
	{"docs", "docs", []string{read, write, delete, thread}},
	{"attachments", "attachments", []string{read, write, delete}},
	{"plays", "plays", []string{read, write, delete, run}},
	{"memories", "memories", []string{read, write, delete, clone}},
	{"tickets", "tickets", []string{read, write, delete}},
	{"deploys", "deploys", []string{read, write}},
	{"stacks", "stacks", []string{read, write, delete}},
	{"topology", "topology", []string{read, write, delete}},
	{"reviews", "code reviews", []string{read}},
	{"repos", "pull requests", []string{read}},
	{"repositories", "repositories", []string{read, write}},
	{"permissions", "permissions", []string{read, write}},
	{"mentions", "mentions", []string{read, write}},
	{"projects", "projects and board settings", []string{read, write, delete}},
	{"workspaces", "workspaces", []string{read, write, delete, create}},
	{"members", "members and invites", []string{read, write, delete}},
	{"roles", "roles", []string{read, write, delete, clone}},
	{"runners", "runners", []string{read, write, delete}},
	{"machines", "machines", []string{read, write}},
	{"dns", "DNS and gateways", []string{read, write, delete}},
	{"notifications", "notifications", []string{read, write}},
	{"chat", "chat", []string{read, write, delete}},
	{"voice", "voice", []string{read, write}},
	{"automations", "automations", []string{read, write, delete}},
	{"integrations", "integrations", []string{read, write, delete}},
	{"connectors", "connectors", []string{read, write}},
	{"events", "events", []string{read}},
	{"audit", "audit log", []string{read}},
	{"accounts", "accounts", []string{read, write, delete}},
	{"instance", "instance settings, upgrades, and failed events", []string{read, write}},
}

// Info is one catalog entry: the action, its owner-facing label, and the domain/action pair grids render from.
type Info struct {
	Value  Action `json:"value"`
	Label  string `json:"label"`
	Domain string `json:"domain"`
	Action string `json:"action"`
}

// Catalog lists every action in display order, so clients render the gate's vocabulary, not their own copy.
func Catalog() []Info {
	out := make([]Info, 0, len(domainTable)*3)
	for _, d := range domainTable {
		for _, a := range d.actions {
			value := Action(d.domain + ":" + a)
			out = append(out, Info{
				Value:  value,
				Label:  label(value, a, d.display),
				Domain: d.domain,
				Action: a,
			})
		}
	}
	return out
}

func label(value Action, a, display string) string {
	switch a {
	case read:
		return "Read " + display
	case write:
		return "Create and update " + display
	case delete:
		return "Delete " + display
	default:
		return verbLabel[value]
	}
}

var known = buildKnown()

func buildKnown() map[Action]bool {
	out := make(map[Action]bool, len(domainTable)*3)
	for _, info := range Catalog() {
		out[info.Value] = true
	}
	return out
}

// AllActions returns every grid action in display order.
func AllActions() []Action {
	catalog := Catalog()
	out := make([]Action, len(catalog))
	for i, info := range catalog {
		out[i] = info.Value
	}
	return out
}

// ParseAction resolves a wire/string action name against the grid.
func ParseAction(s string) (Action, bool) {
	a := Action(strings.TrimSpace(s))
	if !known[a] {
		return "", false
	}
	return a, true
}

// Set is one grant's actions, kept sorted and de-duplicated; it round-trips as a JSON string array.
type Set []Action

// SetOf builds a normalized Set; an empty set is always nil so equality never depends on how it was built.
func SetOf(actions ...Action) Set {
	if len(actions) == 0 {
		return nil
	}
	s := Set(slices.Clone(actions))
	slices.Sort(s)
	return slices.Compact(s)
}

// SetOfStrings builds a Set from wire strings, the shape a permission grid arrives in from another domain.
func SetOfStrings(values []string) Set {
	actions := make([]Action, len(values))
	for i, v := range values {
		actions[i] = Action(v)
	}
	return SetOf(actions...)
}

// CreatorGrant is what a document creator receives: every docs action plus the right to share it.
var CreatorGrant = SetOf(DocsRead, DocsWrite, DocsDelete, PermissionsWrite)

// Has reports whether the set includes action.
func (s Set) Has(action Action) bool {
	_, ok := slices.BinarySearch(s, action)
	return ok
}

// With returns a new set with action added.
func (s Set) With(action Action) Set {
	return SetOf(append(slices.Clone(s), action)...)
}

// Without returns a new set with action removed.
func (s Set) Without(action Action) Set {
	return SetOf(slices.DeleteFunc(slices.Clone(s), func(a Action) bool { return a == action })...)
}

// Except returns the actions in s that other lacks.
func (s Set) Except(other Set) Set {
	return SetOf(slices.DeleteFunc(slices.Clone(s), other.Has)...)
}

// Actions returns the set as a plain slice.
func (s Set) Actions() []Action {
	return slices.Clone(s)
}

// String is the canonical comma-joined action list (for display and logs).
func (s Set) String() string {
	parts := make([]string, len(s))
	for i, a := range s {
		parts[i] = string(a)
	}
	return strings.Join(parts, ",")
}

// MarshalJSON emits an empty set as [] rather than null so stored JSON always parses as an array.
func (s Set) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]Action(s))
}

// UnmarshalJSON normalizes whatever array arrives, so a hand-edited row still behaves as a set.
func (s *Set) UnmarshalJSON(b []byte) error {
	var raw []Action
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*s = SetOf(raw...)
	return nil
}
