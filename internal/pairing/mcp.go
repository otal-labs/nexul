package pairing

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// MCPTools returns the computer tools: pairing, setup runs and confirmations (ADR 0063), MCP tokens, and the skills setup installs.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		computerListTool(s), computerCreateTool(s), computerPairTool(s), computerDeleteTool(s), computerProjectCreateTool(s),
		computerSetupRunTool(s), computerSetupUpdateTool(s), computerMCPTokenCreateTool(s), computerMCPTokenDeleteTool(s),
		skillGetTool(),
	}
}

type computerListIn struct {
	ID            string                   `json:"id,omitempty" jsonschema:"One computer's id, to read it alone with its T3 projects and containers. Omit to list every computer you own."`
	ContainerLogs *computerContainerLogsIn `json:"container_logs,omitzero" jsonschema:"With id only: also return the last lines one Docker container on that computer printed. Omit it for the computer alone."`
	RunnerLogs    *computerRunnerLogsIn    `json:"runner_logs,omitzero" jsonschema:"With id only: also return the last lines the computer's Nexul app (its runner) logged. Omit it for the computer alone."`
	mcptool.PageArgs
}

type computerContainerLogsIn struct {
	Container string `json:"container" jsonschema:"The container's name, from this computer's containers, for example shop-web-1."`
	Tail      int    `json:"tail,omitzero" jsonschema:"How many of the last lines to return, 1 to 1000. Defaults to 200."`
}

type computerRunnerLogsIn struct {
	Lines int `json:"lines,omitzero" jsonschema:"How many of the last lines to return, 1 to 1000. Defaults to 200."`
}

// computerRunnerLogs is the last lines the computer's runner logged.
type computerRunnerLogs struct {
	Lines []containerlog.Line `json:"lines"`
}

// computerContainerLogs is the last lines one container on the computer printed.
type computerContainerLogs struct {
	Container string              `json:"container"`
	Lines     []containerlog.Line `json:"lines"`
}

type computerCreateIn struct {
	Name string `json:"name,omitempty" jsonschema:"The computer's name, for example Alice Laptop. Omit it to name the computer after the hostname it reports when its runner enrolls."`
	ID   string `json:"id,omitempty" jsonschema:"One of your computers that has no runner yet, to get a fresh command for it after the last one expired, or to move a computer paired before the Nexul app onto it. Omit it to add a new computer."`
}

type computerPairIn struct {
	ID   string `json:"id" jsonschema:"One of your computers, from computer_list, to pair or re-pair now through its Nexul app."`
	Name string `json:"name,omitempty" jsonschema:"A new name for the computer, for example Alice Laptop; with it the call only renames the computer. Omit it to pair."`
}

type computerDeleteIn struct {
	ID string `json:"id" jsonschema:"The computer's id, from computer_list."`
}

type computerProjectCreateIn struct {
	ID         string `json:"id" jsonschema:"The computer's id, from computer_list."`
	Repository string `json:"repository,omitempty" jsonschema:"The GitHub repository to clone, as owner/name, for example acme/api; project_get lists a project's repositories. Give this or name, not both."`
	Name       string `json:"name,omitempty" jsonschema:"The folder name of a new empty project, with no slashes, for example acme-notes. Give this or repository, not both."`
}

type computerIDIn struct {
	ComputerID string `json:"computer_id" jsonschema:"The computer's id, from computer_list."`
}

type computerSetupRunIn struct {
	ComputerID   string                             `json:"computer_id" jsonschema:"The paired computer's id, from computer_list."`
	Provider     string                             `json:"provider,omitempty" jsonschema:"One provider's driver kind to run alone, for example codex, after its turn failed. Omit to run every provider the harness lists."`
	Model        string                             `json:"model,omitempty" jsonschema:"With provider only: the model slug its turn runs on, as the harness lists it. Omit for the provider's own default."`
	Providers    []string                           `json:"providers,omitempty" jsonschema:"Without provider only: the driver kinds to set up, for example [\"claude\", \"opencode\"]; the computer remembers the ones left out as skipped. Omit to run every provider the harness lists."`
	Models       map[string]string                  `json:"models,omitempty" jsonschema:"Without provider only: the model slug each provider's turn runs on, keyed by driver kind, for example {\"codex\": \"gpt-5-mini\"}. A provider left out runs on its own default."`
	ModelOptions map[string][]harness.OptionSetting `json:"model_options,omitempty" jsonschema:"The options each provider's picked model runs with, keyed by driver kind like models, for example {\"codex\": [{\"id\": \"reasoningEffort\", \"value\": \"high\"}]}: a choice id for a select option such as reasoning level or context window, true or false for a switch such as fast mode, as the harness lists them per model. Only applies beside a picked model; an option left out keeps the harness default."`
	Folder       string                             `json:"folder,omitempty" jsonschema:"The absolute folder the turns run in, one a T3 Code project on that computer opens, for example /home/me/code/app. Omit for the pairing default, else the harness's first project."`
	SkillsOnly   bool                               `json:"skills_only,omitempty" jsonschema:"true to only update Nexul's skills, alone with computer_id: one turn on the first confirmed provider, on its saved model and folder, for a computer whose providers are confirmed but show skills_outdated."`
}

type computerSetupUpdateIn struct {
	ComputerID    string   `json:"computer_id" jsonschema:"The paired computer's id, from computer_list."`
	Provider      string   `json:"provider,omitempty" jsonschema:"A provider's driver kind, for example claude, codex, or opencode, to change that provider's confirmation. Omit to change the computer's overall confirmation."`
	Confirmed     bool     `json:"confirmed" jsonschema:"true confirms, false withdraws the confirmation."`
	Skills        []string `json:"skills,omitempty" jsonschema:"With provider and confirmed true, required: the skill names this session's harness reported as discovered, for example [\"tdd\", \"nexul-memory\"]."`
	SkillsVersion string   `json:"skills_version,omitempty" jsonschema:"Without provider and with confirmed true: the skills_version skill_get returns, once you have written every Nexul skill from skill_get into every path it lists, for example 3f9a1c0b2d4e. Records it for every confirmed provider on the computer and changes no confirmation."`
}

// computerResult is a computer as an agent reads it: no bearer token, and its setup and MCP token only on a list.
type computerResult struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Kind             harness.Kind           `json:"kind"`
	ServerURL        string                 `json:"server_url"`
	Paired           bool                   `json:"paired"`
	SessionExpiresAt *time.Time             `json:"session_expires_at,omitempty"`
	HarnessVersion   string                 `json:"harness_version,omitempty"`
	Runner           *ComputerRunner        `json:"runner,omitempty"`
	PairError        string                 `json:"pair_error,omitempty"`
	Setup            *Setup                 `json:"setup,omitempty"`
	MCPToken         *MCPToken              `json:"mcp_token,omitempty"`
	T3Projects       []harness.Project      `json:"t3_projects,omitempty"`
	T3ProjectsError  string                 `json:"t3_projects_error,omitempty"`
	Facts            *Facts                 `json:"facts,omitempty"`
	FactsAt          *time.Time             `json:"facts_at,omitempty"`
	Containers       []Container            `json:"containers,omitempty"`
	ContainersError  string                 `json:"containers_error,omitempty"`
	ContainerLogs    *computerContainerLogs `json:"container_logs,omitempty"`
	RunnerLogs       *computerRunnerLogs    `json:"runner_logs,omitempty"`
	Shared           *SharedComputer        `json:"shared,omitempty"`
}

func toComputerResult(c Computer) computerResult {
	r := computerResult{ID: c.ID, Name: c.Name, Kind: c.Kind, ServerURL: c.ServerURL, Paired: c.Paired(), HarnessVersion: c.HarnessVersion, Runner: c.Runner, PairError: c.PairError, Facts: c.Facts, FactsAt: c.FactsAt}
	if r.Paired {
		r.SessionExpiresAt = &c.TokenExpiresAt
	}
	return r
}

func computerListTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_list", "List paired computers",
		"Lists the computers you own, each with its setup confirmation (overall and per provider, plus each "+
			"provider's newest setup turn), and its MCP token's metadata, never the token itself. Pass id to read one "+
			"computer alone with its T3 projects as T3 Code reports them, the t3_project_id play_run takes. A computer "+
			"with paired false has no harness session yet, and pair_error says why its last pairing through its Nexul app "+
			"failed; a computer with no runner was paired before the Nexul app and runs no agent work until computer_create "+
			"with its id moves it. One without setup.confirmed_at needs computer_setup_run "+
			"before agent work can use it. A computer with a runner carries its facts, which only you can read: its system, git "+
			"name and email, free disk, T3 Code's state, install and port, and the providers (version, models, signed in or "+
			"not) and projects T3 Code last listed; facts_at is when they last changed. One computer whose facts show docker "+
			"available also lists its Docker containers (name, image, status); pass container_logs with a container's name "+
			"to read its last lines (200 by default, at most 1000, each cut at 2000 characters), which needs the computer's "+
			"Nexul app connected. Pass runner_logs to read the last lines the computer's Nexul app itself logged, the same way. After yours come the computers other people share with you by name, each named "+
			"\"<owner>'s <computer> (shared)\" with runner.connected and shared (its owner, and the permissions you hold: "+
			"computers:see, computers:run_agents, computers:run_commands, computers:read_logs) and little else; with "+
			"computers:read_logs it also lists the computer's containers and answers container_logs and runner_logs, with "+
			"the container's own env values hidden. Its owner shares or stops sharing one with permission_overwrite_update "+
			"on resource_type computer.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in computerListIn) (any, error) {
			userID := mcpActorID(ctx)
			computers, err := s.ListComputers(ctx, userID)
			if err != nil {
				return nil, err
			}
			if in.ID != "" {
				computers = slices.DeleteFunc(computers, func(c Computer) bool { return c.ID != in.ID })
			}
			if in.ID != "" && len(computers) == 0 {
				return nil, fmt.Errorf("%w: computer %s is not one of yours; computer_list without id lists them", apperrs.ErrNotFound, in.ID)
			}
			if in.ID == "" && (in.ContainerLogs != nil || in.RunnerLogs != nil) {
				return nil, fmt.Errorf("%w: container_logs and runner_logs need the computer's id", apperrs.ErrInvalid)
			}
			out := make([]computerResult, 0, len(computers))
			for _, c := range computers {
				r, err := listedComputer(ctx, s, userID, c, in.ID != "")
				if err != nil {
					return nil, err
				}
				out = append(out, r)
			}
			if in.ContainerLogs != nil {
				if out[0].ContainerLogs, err = containerLogsFor(ctx, s, userID, computers[0], *in.ContainerLogs); err != nil {
					return nil, err
				}
			}
			if in.RunnerLogs != nil {
				if out[0].RunnerLogs, err = runnerLogsFor(ctx, s, userID, computers[0], *in.RunnerLogs); err != nil {
					return nil, err
				}
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})
}

// listedComputer adds the setup and MCP token reads, and the T3 projects and containers only when one computer was asked for.
func listedComputer(ctx context.Context, s *Service, userID string, c Computer, live bool) (computerResult, error) {
	if c.Shared != nil {
		r := computerResult{ID: c.ID, Name: c.SharedName(), Runner: c.Runner, Shared: c.Shared}
		if live && c.Shared.Permissions.Has(permissions.ComputersLogs) {
			r.Containers, r.ContainersError = listedContainers(ctx, s, userID, c)
		}
		return r, nil
	}
	r := toComputerResult(c)
	setup, err := s.GetSetup(ctx, userID, c.ID)
	if err != nil {
		return r, err
	}
	r.Setup = &setup
	if r.MCPToken, err = s.GetMCPToken(ctx, userID, c.ID); err != nil {
		return r, err
	}
	if !live {
		return r, nil
	}
	if r.Paired {
		r.T3Projects, r.T3ProjectsError = listedT3Projects(ctx, s, userID, c.ID)
	}
	if c.Facts != nil && c.Facts.Docker != nil && c.Facts.Docker.Available {
		r.Containers, r.ContainersError = listedContainers(ctx, s, userID, c)
	}
	return r, nil
}

// listedT3Projects asks the computer's harness for its projects; a failure is a line, not the call's.
func listedT3Projects(ctx context.Context, s *Service, userID, computerID string) ([]harness.Project, string) {
	projects, err := s.ListProjects(ctx, userID, computerID)
	if err != nil {
		logging.FromCtx(ctx).Warn("computer t3 projects unavailable", "computer_id", computerID, "err", err)
		return nil, "T3 Code on this computer didn't list its projects; check it is running, then ask again."
	}
	return projects, ""
}

// listedContainers asks the computer's runner for its containers; like the T3 projects, a failure is a line, not the call's.
// A computer whose runner reported no Docker has none to list, which is no failure, for its owner and a sharee alike.
func listedContainers(ctx context.Context, s *Service, userID string, c Computer) ([]Container, string) {
	containers, err := s.ListContainers(ctx, userID, c.ID)
	if err != nil {
		if errors.Is(err, apperrs.ErrConflict) {
			return nil, ""
		}
		logging.FromCtx(ctx).Warn("computer containers unavailable", "computer_id", c.ID, "err", err)
		return nil, fmt.Sprintf("%s didn't list its containers; check its Nexul app is connected, then ask again.", c.Name)
	}
	return containers, ""
}

func containerLogsFor(ctx context.Context, s *Service, userID string, c Computer, in computerContainerLogsIn) (*computerContainerLogs, error) {
	lines, err := s.ContainerLogSnapshot(ctx, userID, c.ID, in.Container, agentTail(in.Tail))
	if err != nil {
		return nil, offlineLogs(err, in.Container, c)
	}
	containerlog.CutForAgent(lines)
	return &computerContainerLogs{Container: in.Container, Lines: lines}, nil
}

func runnerLogsFor(ctx context.Context, s *Service, userID string, c Computer, in computerRunnerLogsIn) (*computerRunnerLogs, error) {
	lines, err := s.RunnerLogSnapshot(ctx, userID, c.ID, agentTail(in.Lines))
	if err != nil {
		return nil, offlineLogs(err, "the Nexul app", c)
	}
	containerlog.CutForAgent(lines)
	return &computerRunnerLogs{Lines: lines}, nil
}

func agentTail(n int) int {
	if n < 1 {
		return containerlog.DefaultTail
	}
	return n
}

// offlineLogs turns the retryable error, which the model would see as an internal one, into the offline case it can act on.
func offlineLogs(err error, what string, c Computer) error {
	if errors.Is(err, apperrs.ErrRetryable) {
		return fmt.Errorf("%w: the logs of %s are unavailable, %s's Nexul app is offline or did not answer; retry once it reconnects",
			apperrs.ErrConflict, what, c.Name)
	}
	return err
}

// computerCreateResult is a computer waiting for its runner, with the command that installs the runner there.
type computerCreateResult struct {
	Computer computerResult `json:"computer"`
	Enrollment
}

func computerCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_create", "Create computer",
		"Adds one of your computers: creates it, waiting for its runner, and returns the one command that installs "+
			"the runner there (commands.unix for Linux and macOS, commands.windows for Windows), ending in one signed token valid "+
			"for an hour. Give the command to the computer's owner to run in a terminal on that computer from their own account; "+
			"on Linux and macOS it uses sudo and installs a service that runs as them, never as root, and on Windows it needs no "+
			"administrator and installs a task that runs as them while they are logged in. Once the "+
			"runner connects, computer_list shows runner.connected on the computer. Pass id instead of name for a "+
			"fresh command when a computer of yours has no runner yet and its last code expired, or to move a computer "+
			"paired before the Nexul app onto it: once its runner connects it pairs through it, keeping its links, setup, "+
			"MCP token and trails. A computer that had a tunnel still runs cloudflared, which sudo cloudflared service "+
			"uninstall (cloudflared service uninstall in an administrator PowerShell on Windows) removes. Only you can see or use the computer.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in computerCreateIn) (any, error) {
			e, err := createComputer(ctx, s, in)
			if err != nil {
				return nil, err
			}
			return computerCreateResult{Computer: toComputerResult(e.Computer), Enrollment: e.Enrollment}, nil
		})
}

func createComputer(ctx context.Context, s *Service, in computerCreateIn) (*ComputerEnrollment, error) {
	if in.ID != "" {
		return s.EnrollComputer(ctx, mcpActorID(ctx), in.ID)
	}
	return s.AddComputer(ctx, mcpActorID(ctx), in.Name)
}

func computerPairTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_pair", "Pair computer",
		"Pairs T3 Code on one of your computers now through its Nexul app, giving Nexul a harness session on it. A "+
			"computer added with computer_create pairs, and re-pairs before its session ends, on its own once its Nexul app "+
			"connects; this pairs it again now, which mints a one-time token on the computer and exchanges it there, and a "+
			"failure says why and leaves the computer as it was. With name it only renames the computer. Returns the paired "+
			"computer; run computer_setup_run next so agent work can use it.",
		mcptool.Hints{},
		func(ctx context.Context, in computerPairIn) (any, error) {
			c, err := pairComputer(ctx, s, in)
			if err != nil {
				return nil, err
			}
			return toComputerResult(*c), nil
		})
}

func pairComputer(ctx context.Context, s *Service, in computerPairIn) (*Computer, error) {
	if in.Name != "" {
		return s.RenameComputer(ctx, mcpActorID(ctx), in.ID, in.Name)
	}
	return s.PairComputer(ctx, mcpActorID(ctx), in.ID)
}

func computerDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_delete", "Delete computer",
		"Removes one of your computers: retires its Nexul app, which removes itself from the computer, revokes its MCP "+
			"token, then deletes it. Agent work can no longer run there; add it again with computer_create. A failed step "+
			"keeps the computer, so calling this again retries it. Returns {id, deleted: true}.",
		mcptool.Hints{Idempotent: true},
		func(ctx context.Context, in computerDeleteIn) (any, error) {
			if err := s.DeleteComputer(ctx, mcpActorID(ctx), in.ID); err != nil {
				return nil, err
			}
			return mcptool.Gone(in.ID), nil
		})
}

func computerProjectCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_project_create", "Add a project to a computer",
		"Adds a new T3 project to one of your paired computers so plays and @Agent turns can run in it there; use it "+
			"when computer_list with the computer's id lists no T3 project for the work. With repository, a GitHub "+
			"repository as owner/name, the computer clones it over https with its own git credentials; with name it makes "+
			"an empty folder, for a project with no repository or a computer with no T3 project at all, which setup also "+
			"needs. Either lands in the folder most of the computer's T3 projects share, or ~/Code when it has none. "+
			"Returns the new project's id, the t3_project_id play_run takes; a clone keeps running in the background, and "+
			"one that fails leaves the project in T3 Code to retry or remove there. A clone fails when the folder "+
			"already exists and is not empty, and on a computer paired before cloning existed until computer_pair pairs "+
			"it again; an empty project does not need that.",
		mcptool.Hints{Additive: true},
		func(ctx context.Context, in computerProjectCreateIn) (any, error) {
			return s.AddProject(ctx, mcpActorID(ctx), in.ID, in.Repository, in.Name)
		})
}

func computerSetupRunTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_setup_run", "Run computer setup",
		"Starts setup on one of your paired computers: Nexul mints or reuses the computer's MCP token, then runs one "+
			"setup turn per provider its harness lists, one after another, each connecting Nexul's MCP server, "+
			"installing the default skills and nexul-memory, and confirming through computer_setup_update. Pass "+
			"providers to set up only some of them (the rest are remembered as skipped), or "+
			"provider to run only that provider's turn again after it failed; on a confirmed provider it re-verifies "+
			"without undoing anything. Pass skills_only when computer_list shows skills_outdated on confirmed providers "+
			"and nothing else needs setup: one short turn on the first confirmed provider rewrites Nexul's skills from "+
			"skill_get into the folders every provider reads and reports the version for all of them, with no MCP "+
			"reconnect and no per-provider check. Returns at once with the run and its providers; computer_list shows each "+
			"provider's turn state as it progresses. Setup only writes user-level files, so pass folder to run it in any "+
			"project folder when the default one is gone. Only one setup or skills update runs on a computer at a time.",
		mcptool.Hints{},
		func(ctx context.Context, in computerSetupRunIn) (any, error) {
			if in.SkillsOnly {
				return updateSkills(ctx, s, in)
			}
			if in.Provider == "" && in.Model != "" {
				return nil, fmt.Errorf("%w: model applies to one provider; pass provider too, or models to pick per provider", apperrs.ErrInvalid)
			}
			if in.Provider != "" && len(in.Models) > 0 {
				return nil, fmt.Errorf("%w: models applies to a run of every provider; with provider pass model instead", apperrs.ErrInvalid)
			}
			if in.Provider != "" && len(in.Providers) > 0 {
				return nil, fmt.Errorf("%w: providers applies to a run of several providers; with provider pass just that one", apperrs.ErrInvalid)
			}
			if in.Provider == "" {
				return s.StartSetup(ctx, mcpActorID(ctx), in.ComputerID, in.Models, in.ModelOptions, in.Folder, in.Providers)
			}
			return s.RetrySetupProvider(ctx, mcpActorID(ctx), in.ComputerID, in.Provider, in.Model, in.ModelOptions[in.Provider], in.Folder)
		})
}

// updateSkills is computer_setup_run's skills_only mode, which picks its own provider, model, and folder.
func updateSkills(ctx context.Context, s *Service, in computerSetupRunIn) (*SetupRun, error) {
	if in.Provider != "" || in.Model != "" || len(in.Providers) > 0 || len(in.Models) > 0 || len(in.ModelOptions) > 0 || in.Folder != "" {
		return nil, fmt.Errorf("%w: skills_only goes alone with computer_id; it runs on the first confirmed provider with its saved model and folder", apperrs.ErrInvalid)
	}
	return s.UpdateSkills(ctx, mcpActorID(ctx), in.ComputerID)
}

func computerSetupUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_setup_update", "Update setup confirmation",
		"Confirms or withdraws a setup confirmation on one of your paired computers; agent work runs there only while "+
			"both the computer's overall confirmation and its provider's are set. With provider and confirmed true, "+
			"it records that Nexul's MCP server is connected to that provider and its harness reports the given "+
			"skills; without provider it confirms the computer overall, the last step of its setup. Confirmed false "+
			"withdraws the one named, and withdrawing the overall confirmation also revokes the computer's MCP token. "+
			"With skills_version and confirmed true, alone, it records the skills_version you just wrote from "+
			"skill_get for every confirmed provider, since they read the same skill folders, and confirms nothing. "+
			"Only this tool writes a confirmation, and repeating a call changes nothing more; returns the computer's "+
			"setup, which computer_list also shows.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in computerSetupUpdateIn) (any, error) {
			if in.SkillsVersion != "" && (in.Provider != "" || len(in.Skills) > 0 || !in.Confirmed) {
				return nil, fmt.Errorf("%w: skills_version goes alone with computer_id and confirmed true", apperrs.ErrInvalid)
			}
			if in.SkillsVersion != "" {
				return s.RecordSkillsVersion(ctx, mcpActorID(ctx), in.ComputerID, in.SkillsVersion)
			}
			if len(in.Skills) > 0 && (in.Provider == "" || !in.Confirmed) {
				return nil, fmt.Errorf("%w: skills go only with provider and confirmed true", apperrs.ErrInvalid)
			}
			return updateSetup(ctx, s, mcpActorID(ctx), in)
		})
}

func updateSetup(ctx context.Context, s *Service, userID string, in computerSetupUpdateIn) (Setup, error) {
	if in.Provider == "" && in.Confirmed {
		return s.ConfirmSetup(ctx, userID, in.ComputerID)
	}
	if in.Provider == "" {
		return s.UnconfirmSetup(ctx, userID, in.ComputerID)
	}
	if in.Confirmed {
		return s.ConfirmProviderSetup(ctx, userID, in.ComputerID, in.Provider, in.Skills)
	}
	return s.UnconfirmProviderSetup(ctx, userID, in.ComputerID, in.Provider)
}

func computerMCPTokenCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_mcp_token_create", "Create computer MCP token",
		"Mints one of your paired computers its own personal access token, \"Nexul MCP on <computer>\", for its "+
			"providers' MCP configs, revoking the one it had. The token is returned only in this response and is "+
			"hidden in saved transcripts, so show it only to the computer's owner; computer_list shows its metadata "+
			"afterwards. computer_setup_run mints one on its own, so call this only to rotate the token by hand.",
		mcptool.Hints{Local: true},
		func(ctx context.Context, in computerIDIn) (any, error) {
			return s.MintMCPToken(ctx, mcpActorID(ctx), in.ComputerID)
		})
}

func computerMCPTokenDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_mcp_token_delete", "Delete computer MCP token",
		"Revokes one of your paired computers' MCP token, so its providers lose Nexul's MCP server until "+
			"computer_mcp_token_create or computer_setup_run mints a new one. Returns the revoked token's id. "+
			"A computer without a token is not found; computer_list shows which computers have one.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in computerIDIn) (any, error) {
			userID := mcpActorID(ctx)
			token, err := s.GetMCPToken(ctx, userID, in.ComputerID)
			if err != nil {
				return nil, err
			}
			if token == nil {
				return nil, fmt.Errorf("%w: computer %s has no MCP token; computer_mcp_token_create mints one", apperrs.ErrNotFound, in.ComputerID)
			}
			if err := s.RevokeMCPToken(ctx, userID, in.ComputerID); err != nil {
				return nil, err
			}
			return mcptool.Gone(token.ID), nil
		})
}

func mcpActorID(ctx context.Context) string {
	if a, ok := identity.ActorFromCtx(ctx); ok {
		return a.ID
	}
	return ""
}
