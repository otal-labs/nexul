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
	DocsClone          Action = "docs:clone"
	DocsLock           Action = "docs:lock"
	PermissionsWrite   Action = "permissions:write"
	ProjectsRead       Action = "projects:read"
	ProjectsWrite      Action = "projects:write"
	ProjectsDelete     Action = "projects:delete"
	ProjectsCreate     Action = "projects:create"
	ProjectsAccess     Action = "projects:access"
	TicketsRead        Action = "tickets:read"
	TicketsWrite       Action = "tickets:write"
	TicketsDelete      Action = "tickets:delete"
	StacksRead         Action = "stacks:read"
	StacksWrite        Action = "stacks:write"
	StacksDelete       Action = "stacks:delete"
	StacksLogs         Action = "stacks:logs"
	DeploysRead        Action = "deploys:read"
	DeploysWrite       Action = "deploys:write"
	TopologyRead       Action = "topology:read"
	TopologyWrite      Action = "topology:write"
	TopologyDelete     Action = "topology:delete"
	MachinesWrite      Action = "machines:write"
	MachinesDeploy     Action = "machines:deploy"
	MachinesDelete     Action = "machines:delete"
	MachinesCreate     Action = "machines:create"
	DNSRead            Action = "dns:read"
	DNSWrite           Action = "dns:write"
	DNSDelete          Action = "dns:delete"
	ReviewsRead        Action = "reviews:read"
	ReposRead          Action = "repos:read"
	ConnectorsRead     Action = "connectors:read"
	ConnectorsWrite    Action = "connectors:write"
	ChatWrite          Action = "chat:write"
	ChannelsRead       Action = "channels:read"
	ChannelsWrite      Action = "channels:write"
	ChannelsDelete     Action = "channels:delete"
	ChannelsPost       Action = "channels:post"
	BotwebhookRead     Action = "botwebhook:read"
	BotwebhookWrite    Action = "botwebhook:write"
	BotwebhookDelete   Action = "botwebhook:delete"
	WorkspacesWrite    Action = "workspaces:write"
	WorkspacesDelete   Action = "workspaces:delete"
	WorkspacesCreate   Action = "workspaces:create"
	MembersWrite       Action = "members:write"
	MembersViewAs      Action = "members:view_as"
	RolesWrite         Action = "roles:write"
	RolesClone         Action = "roles:clone"
	AutomationsRead    Action = "automations:read"
	AutomationsWrite   Action = "automations:write"
	AutomationsDelete  Action = "automations:delete"
	PlaysRead          Action = "plays:read"
	PlaysWrite         Action = "plays:write"
	PlaysDelete        Action = "plays:delete"
	PlaysRun           Action = "plays:run"
	AutoplaysRead      Action = "autoplays:read"
	AutoplaysWrite     Action = "autoplays:write"
	AutoplaysDelete    Action = "autoplays:delete"
	MemoriesRead       Action = "memories:read"
	MemoriesWrite      Action = "memories:write"
	MemoriesDelete     Action = "memories:delete"
	MemoriesClone      Action = "memories:clone"
	MemoriesTemplate   Action = "memories:template"
	InstanceRead       Action = "instance:read"
	InstanceWrite      Action = "instance:write"
	AccountsRead       Action = "accounts:read"
	AccountsWrite      Action = "accounts:write"
	AccountsDelete     Action = "accounts:delete"
	IntegrationsRead   Action = "integrations:read"
	IntegrationsWrite  Action = "integrations:write"
	IntegrationsDelete Action = "integrations:delete"
	AuditRead          Action = "audit:read"
	LogsRead           Action = "logs:read"
	EventsRead         Action = "events:read"
	TemplatesRead      Action = "templates:read"
	TemplatesWrite     Action = "templates:write"
)

// ProjectsOpen is In this project: held only through a project's rules, never by a role or a token, so it is off the
// grid and ParseAction refuses it (ADR 0148).
const ProjectsOpen Action = "projects:open"

// MachinesSee is See on a machine: held only through the machine's rules, never by a role or a token, so it is off the
// grid and ParseAction refuses it (ADR 0151).
const MachinesSee Action = "machines:see"

// MachineActions are what a machine's rules may set, See first; Manage is machines:write.
var MachineActions = []Action{MachinesSee, MachinesDeploy, MachinesWrite, MachinesDelete}

// The computer permissions, off the grid like ProjectsOpen: held only through a rule a computer's owner writes for a
// named person, never by a role, an invitation or a token (ADR 0148).
const (
	ComputersSee      Action = "computers:see"
	ComputersAgents   Action = "computers:run_agents"
	ComputersCommands Action = "computers:run_commands"
	ComputersLogs     Action = "computers:read_logs"
)

// ComputerActions are what a computer rule may set, See first.
var ComputerActions = []Action{ComputersSee, ComputersAgents, ComputersCommands, ComputersLogs}

const (
	read     = "read"
	write    = "write"
	delete   = "delete"
	run      = "run"
	clone    = "clone"
	thread   = "thread"
	lock     = "lock"
	create   = "create"
	logs     = "logs"
	access   = "access"
	template = "template"
	post     = "post"
	deploy   = "deploy"
	viewAs   = "view_as"
)

// verbLabel is the owner-facing text for a domain-declared verb (ADR 0057): one line per verb, no per-domain switch.
var verbLabel = map[Action]string{
	PlaysRun:         "Run plays",
	MemoriesClone:    "Clone memories to another project",
	MemoriesTemplate: "Edit the workspace Interview template",
	RolesClone:       "Clone roles to another workspace",
	DocsThread:       "See doc threads",
	DocsClone:        "Clone docs into another project",
	DocsLock:         "Lock and unlock docs",
	StacksLogs:       "Read container logs",
	// The creator owns what they create, and an Owner holds every permission, so this one is as strong as Owner.
	WorkspacesCreate: "Create workspaces",
	ProjectsCreate:   "Create projects",
	ProjectsAccess:   "Manage this project's access",
	ChannelsPost:     "Post in channels",
	MachinesDeploy:   "Deploy to machines",
	MachinesWrite:    "Manage machines",
	MachinesCreate:   "Add machines",
	MembersViewAs:    "View Nexul as another member",
}

// Area is where a domain's permission applies (ADR 0097): inside one project, across the workspace, or instance-wide.
type Area string

// The three areas; a project-area action is held only inside a project its holder is in (ADR 0148).
const (
	AreaProject   Area = "project"
	AreaWorkspace Area = "workspace"
	AreaInstance  Area = "instance"
)

// domainInfo is one row of the grid: an API domain, its display name, the actions its routes expose, and its area.
type domainInfo struct {
	domain  string
	display string
	actions []string
	area    Area
}

// domainTable is the single source every catalog, valid-action set, and web grid derives from (display order).
var domainTable = []domainInfo{
	{"docs", "docs", []string{read, write, delete, thread, clone, lock}, AreaProject},
	{"attachments", "attachments", []string{read, write, delete}, AreaProject},
	{"plays", "plays", []string{read, write, delete, run}, AreaWorkspace},
	{"autoplays", "auto plays", []string{read, write, delete}, AreaWorkspace},
	{"memories", "memories", []string{read, write, delete, clone, template}, AreaProject},
	{"tickets", "tickets", []string{read, write, delete}, AreaProject},
	{"deploys", "deploys", []string{read, write}, AreaProject},
	{"stacks", "stacks", []string{read, write, delete, logs}, AreaProject},
	{"topology", "topology", []string{read, write, delete}, AreaInstance},
	{"reviews", "code reviews", []string{read}, AreaProject},
	{"repos", "pull requests", []string{read}, AreaProject},
	{"repositories", "repositories", []string{read, write}, AreaProject},
	{"permissions", "permissions", []string{read, write}, AreaProject},
	{"mentions", "mentions", []string{read, write}, AreaWorkspace},
	{"projects", "projects and board settings", []string{read, write, delete, create, access}, AreaProject},
	{"workspaces", "workspaces", []string{read, write, delete, create}, AreaWorkspace},
	{"members", "members and invites", []string{read, write, delete, viewAs}, AreaWorkspace},
	{"roles", "roles", []string{read, write, delete, clone}, AreaWorkspace},
	{"machines", "machines", []string{write, deploy, delete, create}, AreaInstance},
	{"dns", "DNS and gateways", []string{read, write, delete}, AreaInstance},
	{"notifications", "notifications", []string{read, write}, AreaWorkspace},
	{"chat", "chat", []string{read, write, delete}, AreaWorkspace},
	// A channel's read starts held for whoever may see it and only a rule denies it; posting takes channels:post (ADR 0150).
	{"channels", "channels", []string{read, write, delete, post}, AreaWorkspace},
	{"voice", "voice", []string{read, write}, AreaWorkspace},
	// No implication between the three: read hides a bot's URL, write shows it, and only delete deletes.
	{"botwebhook", "bots", []string{read, write, delete}, AreaWorkspace},
	{"automations", "automations", []string{read, write, delete}, AreaWorkspace},
	{"integrations", "integrations", []string{read, write, delete}, AreaInstance},
	{"connectors", "connectors", []string{read, write}, AreaInstance},
	{"events", "events", []string{read}, AreaInstance},
	{"audit", "audit log", []string{read}, AreaInstance},
	{"logs", "the instance's logs", []string{read}, AreaInstance},
	{"accounts", "accounts", []string{read, write, delete}, AreaInstance},
	{"instance", "instance settings, upgrades, and failed events", []string{read, write}, AreaInstance},
	// templates:read scopes a token's reads; a signed-in member reads instance templates with membership alone.
	{"templates", "instance templates", []string{read, write}, AreaInstance},
}

// InstanceArea is one instance area a rule may sit on (ADR 0148): its key, its name in a sentence, and the actions its
// rules may set.
type InstanceArea struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Actions []Action `json:"actions"`
}

// SignInProviders is the sign-in providers' area, which shares instance:read and instance:write with the instance's own.
const SignInProviders = "sign_in_providers"

// instanceAreas lists the fixed instance areas in display order; ConnectorArea adds one per connector.
var instanceAreas = []InstanceArea{
	{"instance", "the instance", []Action{InstanceRead, InstanceWrite, EventsRead}},
	{SignInProviders, "sign-in providers", []Action{InstanceRead, InstanceWrite}},
	{"dns", "DNS and gateways", []Action{DNSRead, DNSWrite, DNSDelete}},
	{"templates", "instance templates", []Action{TemplatesRead, TemplatesWrite}},
	{"accounts", "accounts", []Action{AccountsRead, AccountsWrite, AccountsDelete}},
	{"machines", "adding machines", []Action{MachinesCreate}},
	{"topology", "the topology registry", []Action{TopologyRead, TopologyWrite, TopologyDelete}},
	{"integrations", "integrations", []Action{IntegrationsRead, IntegrationsWrite, IntegrationsDelete}},
	{"audit", "the audit log", []Action{AuditRead}},
	{"logs", "the instance's logs", []Action{LogsRead}},
}

// InstanceAreas is the fixed instance areas, without the connectors'.
func InstanceAreas() []InstanceArea {
	return slices.Clone(instanceAreas)
}

// ConnectorArea is the area of the connector with id, named name: each connector registers its own.
func ConnectorArea(id, name string) InstanceArea {
	return InstanceArea{Key: ConnectorAreaKey(id), Label: "connector " + name, Actions: []Action{ConnectorsRead, ConnectorsWrite}}
}

// ConnectorAreaKey is the key of the connector with id's area.
func ConnectorAreaKey(id string) string {
	return "connector:" + id
}

// InstanceAreaOf is the area a check of action asks about when it names none: the first fixed area listing it, ""
// for an action outside every fixed area, which a connector's check names itself.
func InstanceAreaOf(action Action) string {
	for _, a := range instanceAreas {
		if slices.Contains(a.Actions, action) {
			return a.Key
		}
	}
	return ""
}

// actionArea is the area of an action whose area differs from its domain's: creating a project, or editing the
// workspace's Interview template, happens outside one.
var actionArea = map[Action]Area{ProjectsCreate: AreaWorkspace, MemoriesTemplate: AreaWorkspace}

// Info is one catalog entry: the action, its owner-facing label, and the domain/action pair grids render from.
type Info struct {
	Value  Action `json:"value"`
	Label  string `json:"label"`
	Domain string `json:"domain"`
	Action string `json:"action"`
	Area   Area   `json:"area"`
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
				Area:   areaOf(value, d.area),
			})
		}
	}
	return out
}

func label(value Action, a, display string) string {
	if l, ok := verbLabel[value]; ok {
		return l
	}
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

var areas = buildAreas()

func buildAreas() map[string]Area {
	out := make(map[string]Area, len(domainTable))
	for _, d := range domainTable {
		out[d.domain] = d.area
	}
	return out
}

// AreaOf is the area of action, its domain's unless the action has its own; the empty Member action and anything off
// the grid are workspace-wide.
func AreaOf(action Action) Area {
	domain, _, _ := strings.Cut(string(action), ":")
	if area, ok := areas[domain]; ok {
		return areaOf(action, area)
	}
	return AreaWorkspace
}

func areaOf(action Action, domainArea Area) Area {
	if area, ok := actionArea[action]; ok {
		return area
	}
	return domainArea
}

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

// ParseRuleAction is ParseAction for a rule on an entity, which may also set what only a rule holds: In this project,
// See on a machine and the computer permissions.
func ParseRuleAction(s string) (Action, bool) {
	if a := Action(strings.TrimSpace(s)); a == ProjectsOpen || a == MachinesSee || slices.Contains(ComputerActions, a) {
		return a, true
	}
	return ParseAction(s)
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

// CreatorGrant is what a document creator receives: reading, editing, and deleting it, plus the right to share it;
// locking it comes only from their roles.
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
