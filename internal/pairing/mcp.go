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
	ID            string                   `json:"id,omitempty" jsonschema:"One computer's id, to read it alone with its T3 projects and containers. Omit to list all."`
	ContainerLogs *computerContainerLogsIn `json:"container_logs,omitzero" jsonschema:"With id only: also returns one Docker container's last log lines."`
	RunnerLogs    *computerRunnerLogsIn    `json:"runner_logs,omitzero" jsonschema:"With id only: also returns the Nexul app's (the runner's) last log lines."`
	mcptool.PageArgs
}

type computerContainerLogsIn struct {
	Container string `json:"container" jsonschema:"The container's name, from this computer's containers."`
	Tail      int    `json:"tail,omitzero" jsonschema:"Last lines to return, 1 to 1000. Defaults to 200."`
}

type computerRunnerLogsIn struct {
	Lines int `json:"lines,omitzero" jsonschema:"Last lines to return, 1 to 1000. Defaults to 200."`
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
	Name string `json:"name,omitempty" jsonschema:"The computer's name. Omit to use the hostname its runner reports."`
	ID   string `json:"id,omitempty" jsonschema:"One of your computers with no runner yet, for a fresh command after the last expired, or to move a computer paired before the Nexul app onto it. Omit to add a new computer."`
}

type computerPairIn struct {
	ID            string  `json:"id" jsonschema:"One of your computers, from computer_list."`
	Name          string  `json:"name,omitempty" jsonschema:"A new name. With it, or any field below, the call only changes those and does not pair."`
	Position      *int    `json:"position,omitempty" jsonschema:"Place in your order, 0 for first: runs go to the first computer that can take them."`
	RunsAtOnce    *int    `json:"runs_at_once,omitempty" jsonschema:"Play runs the computer takes at a time, 1 to 5; a queued auto run waits while every computer is full."`
	ChatProjectID *string `json:"chat_project_id,omitempty" jsonschema:"The T3 project, from computer_list with this computer's id, that @Agent chats outside any project run in; empty clears it."`
}

type computerDeleteIn struct {
	ID string `json:"id" jsonschema:"The computer's id, from computer_list."`
}

type computerProjectCreateIn struct {
	ID         string `json:"id" jsonschema:"The computer's id, from computer_list."`
	Repository string `json:"repository,omitempty" jsonschema:"The GitHub repository to clone, as owner/name; project_get lists a project's repositories. Give this or name, not both."`
	Name       string `json:"name,omitempty" jsonschema:"The folder name of a new empty project, with no slashes. Give this or repository, not both."`
}

type computerIDIn struct {
	ComputerID string `json:"computer_id" jsonschema:"The computer's id, from computer_list."`
}

type computerSetupRunIn struct {
	ComputerID   string                             `json:"computer_id" jsonschema:"The paired computer's id, from computer_list."`
	Provider     string                             `json:"provider,omitempty" jsonschema:"One provider's driver kind, for example codex, to run alone after its turn failed. Omit to run every provider."`
	Model        string                             `json:"model,omitempty" jsonschema:"With provider only: the model slug, as the harness lists it. Omit for the provider's default."`
	Providers    []string                           `json:"providers,omitempty" jsonschema:"Without provider only: the driver kinds to set up, for example [\"claude\", \"opencode\"]; the rest are remembered as skipped. Omit to run every provider."`
	Models       map[string]string                  `json:"models,omitempty" jsonschema:"Without provider only: the model slug per provider, keyed by driver kind, for example {\"codex\": \"gpt-5-mini\"}. A provider left out uses its default."`
	ModelOptions map[string][]harness.OptionSetting `json:"model_options,omitempty" jsonschema:"Options each provider's picked model runs with, keyed by driver kind like models, for example {\"codex\": [{\"id\": \"reasoningEffort\", \"value\": \"high\"}]}: a choice id for a select option, true or false for a switch, as the harness lists them per model. Only applies beside a picked model."`
	Folder       string                             `json:"folder,omitempty" jsonschema:"The absolute folder the turns run in, one a T3 Code project on that computer opens, for example /home/me/code/app. Omit for the pairing default."`
	SkillsOnly   bool                               `json:"skills_only,omitempty" jsonschema:"true only updates Nexul's skills and goes alone with computer_id: one turn on the first confirmed provider, on its saved model and folder."`
	Stop         bool                               `json:"stop,omitempty" jsonschema:"true with provider stops that provider's running setup or skills turn; it ends failed as Stopped."`
}

type computerSetupUpdateIn struct {
	ComputerID    string   `json:"computer_id" jsonschema:"The paired computer's id, from computer_list."`
	Provider      string   `json:"provider,omitempty" jsonschema:"A provider's driver kind, for example claude, codex, or opencode, to change that provider's confirmation. Omit for the computer's overall confirmation."`
	Confirmed     bool     `json:"confirmed" jsonschema:"true confirms, false withdraws."`
	Skills        []string `json:"skills,omitempty" jsonschema:"With provider and confirmed true, required: the skill names this session's harness reported as discovered, for example [\"tdd\", \"nexul-memory\"]."`
	SkillsVersion string   `json:"skills_version,omitempty" jsonschema:"Without provider and with confirmed true: the skills_version skill_get returns, once every Nexul skill is written into every path it lists. Recorded for every confirmed provider."`
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
	Position         int                    `json:"position"`
	RunsAtOnce       int                    `json:"runs_at_once"`
	Running          int                    `json:"running"`
	ChatProjectID    string                 `json:"chat_project_id,omitempty"`
}

func toComputerResult(c Computer) computerResult {
	r := computerResult{
		ID: c.ID, Name: c.Name, Kind: c.Kind, ServerURL: c.ServerURL, Paired: c.Paired(), HarnessVersion: c.HarnessVersion, Runner: c.Runner,
		PairError: c.PairError, Facts: c.Facts, FactsAt: c.FactsAt, Position: c.Position, RunsAtOnce: c.RunsAtOnce, Running: c.Running,
		ChatProjectID: c.ChatProjectID,
	}
	if r.Paired {
		r.SessionExpiresAt = &c.TokenExpiresAt
	}
	return r
}

func computerListTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_list", "List paired computers",
		"Lists the computers you own in your order (runs go to the first that can take them), each with its "+
			"position, runs_at_once, running count, chat_project_id, setup confirmation (overall, per provider, newest "+
			"setup turn) and its MCP token's metadata, never the token. Pass id to read one with its T3 projects, "+
			"the t3_project_id play_run takes. paired false means no harness session yet (pair_error says why); "+
			"no runner means it predates the Nexul app and runs no agent work until computer_create with its id "+
			"moves it; no setup.confirmed_at needs computer_setup_run first. A computer with a runner carries facts "+
			"only you can read. With id and docker in its facts it also lists Docker containers; "+
			"container_logs reads a container's last lines and runner_logs the Nexul app's own (each line cut at 2000 "+
			"characters), both needing its Nexul app connected. Computers shared with you follow, named "+
			"\"<owner>'s <computer> (shared)\", with runner state and shared (owner and your permissions: "+
			"computers:see, computers:run_agents, computers:run_commands, computers:read_logs); computers:read_logs "+
			"adds containers and both logs, container env values hidden. Its owner shares one with "+
			"permission_overwrite_update on resource_type computer.",
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
		"Creates one of your computers, waiting for its runner, and returns the one command that installs "+
			"the runner there (commands.unix for Linux and macOS, commands.windows for Windows), with a signed token valid "+
			"for an hour. The computer's owner runs it from their own account in a terminal there: "+
			"on Linux and macOS it uses sudo and installs a service running as them, never root; on Windows it needs no "+
			"administrator. Once the runner connects, computer_list shows "+
			"runner.connected. Pass id instead of name for a fresh command when a computer has no runner yet and its "+
			"last code expired, or to move a computer paired before the Nexul app onto it, keeping its links, setup, "+
			"MCP token and trails. A computer that had a tunnel still runs cloudflared; sudo cloudflared service "+
			"uninstall removes it (in an administrator PowerShell on Windows, without sudo).",
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
		"Pairs T3 Code on one of your computers now through its Nexul app, minting a one-time token there and "+
			"exchanging it for a harness session. A computer from computer_create pairs, and re-pairs before its "+
			"session ends, on its own once its Nexul app connects. With name, position, runs_at_once or "+
			"chat_project_id it only changes those and pairs nothing. Returns the computer; after pairing, run "+
			"computer_setup_run next.",
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
	userID := mcpActorID(ctx)
	if in.Name == "" && in.Position == nil && in.RunsAtOnce == nil && in.ChatProjectID == nil {
		return s.PairComputer(ctx, userID, in.ID)
	}
	if in.Position != nil {
		if err := s.MoveComputer(ctx, userID, in.ID, *in.Position); err != nil {
			return nil, err
		}
	}
	patch := ComputerPatch{RunsAtOnce: in.RunsAtOnce, ChatProjectID: in.ChatProjectID}
	if in.Name != "" {
		patch.Name = &in.Name
	}
	return s.UpdateComputer(ctx, userID, in.ID, patch)
}

func computerDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_delete", "Delete computer",
		"Removes one of your computers: retires its Nexul app, which removes itself from the computer, revokes its MCP "+
			"token, then deletes it. Agent work can no longer run there; add it again with computer_create. A failed step "+
			"keeps the computer, so calling this again retries it.",
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
		"Adds a T3 project to one of your paired computers so plays and @Agent turns can run in it; use it "+
			"when computer_list with the computer's id lists none for the work. With repository the Nexul app there "+
			"clones it with a read-only token, so the computer needs no GitHub sign-in (an older app, or no GitHub App "+
			"key, leaves it to the computer's own git credentials); with name it makes an empty folder, which a "+
			"computer with no T3 project needs for setup. Either lands in the folder most of the computer's T3 "+
			"projects share, or ~/Code when it has none. Returns the new project's id, the t3_project_id play_run "+
			"takes, once the clone is done; a failed clone leaves no project behind, and on a computer paired before "+
			"cloning existed it fails until computer_pair pairs it again.",
		mcptool.Hints{Additive: true},
		func(ctx context.Context, in computerProjectCreateIn) (any, error) {
			return s.AddProject(ctx, mcpActorID(ctx), in.ID, in.Repository, in.Name)
		})
}

func computerSetupRunTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_setup_run", "Run computer setup",
		"Starts setup on one of your paired computers: mints or reuses its MCP token, then runs one "+
			"setup turn per provider its harness lists, each connecting Nexul's MCP server, "+
			"installing the default skills and nexul-memory, and confirming through computer_setup_update. Pass "+
			"providers to set up only some (the rest are remembered as skipped), or provider to rerun one "+
			"provider's turn after it failed; on a confirmed provider that re-verifies. Pass skills_only when "+
			"computer_list shows skills_outdated on confirmed providers: one short turn on the first confirmed "+
			"provider rewrites Nexul's skills from skill_get and reports the version for all. Returns at once with "+
			"the run; computer_list shows each turn's state. Pass folder to run in any project folder when the "+
			"default is gone. One setup or skills update runs per computer at a time. Pass stop with provider to stop "+
			"that provider's running turn.",
		mcptool.Hints{},
		func(ctx context.Context, in computerSetupRunIn) (any, error) {
			if in.Stop {
				return stopSetup(ctx, s, in)
			}
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

// stopSetup is computer_setup_run's stop mode, which ends one provider's running turn.
func stopSetup(ctx context.Context, s *Service, in computerSetupRunIn) (any, error) {
	if in.Provider == "" || in.SkillsOnly || in.Model != "" || len(in.Providers) > 0 || len(in.Models) > 0 || len(in.ModelOptions) > 0 || in.Folder != "" {
		return nil, fmt.Errorf("%w: stop goes with computer_id and provider only", apperrs.ErrInvalid)
	}
	if err := s.StopSetupProvider(ctx, mcpActorID(ctx), in.ComputerID, in.Provider); err != nil {
		return nil, err
	}
	return map[string]any{"stopped": in.Provider}, nil
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
			"both its overall confirmation and its provider's are set. With provider and confirmed true it records "+
			"that Nexul's MCP server is connected to that provider and its harness reports the given skills; "+
			"without provider it confirms the computer overall, the last step of setup. Confirmed false "+
			"withdraws the one named, and withdrawing the overall confirmation also revokes the computer's MCP token. "+
			"With skills_version and confirmed true, alone, it records that version for every confirmed provider "+
			"and confirms nothing. Returns the computer's setup.",
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
		"Mints one of your paired computers a personal access token, \"Nexul MCP on <computer>\", for its "+
			"providers' MCP configs, revoking the one it had. The token is returned only in this response, hidden "+
			"in saved transcripts, so show it only to the computer's owner. computer_setup_run mints one on its "+
			"own, so call this only to rotate the token by hand.",
		mcptool.Hints{Local: true},
		func(ctx context.Context, in computerIDIn) (any, error) {
			return s.MintMCPToken(ctx, mcpActorID(ctx), in.ComputerID)
		})
}

func computerMCPTokenDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("computer_mcp_token_delete", "Delete computer MCP token",
		"Revokes one of your paired computers' MCP token, so its providers lose Nexul's MCP server until "+
			"computer_mcp_token_create or computer_setup_run mints a new one. Returns the revoked token's id. "+
			"computer_list shows which computers have one.",
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
