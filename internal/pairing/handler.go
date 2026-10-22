package pairing

import (
	"errors"
	"net/http"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/containerlog"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// Handler adapts the pairing use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the pairing REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the pairing REST endpoints, mounted behind RequireAuth at /api/pairing.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/pairing/computers", h.listComputers)
	mux.HandleFunc("POST /api/pairing/computers", retired)
	mux.HandleFunc("POST /api/pairing/computers/enrollments", h.enrollComputer)
	mux.HandleFunc("PATCH /api/pairing/computers/{id}", h.updateComputer)
	mux.HandleFunc("PUT /api/pairing/computers/order", h.reorderComputers)
	mux.HandleFunc("POST /api/pairing/computers/tunnel", retired)
	mux.HandleFunc("GET /api/pairing/computers/{id}/tunnel/status", retired)
	mux.HandleFunc("GET /api/pairing/computers/{id}/tunnel/token", retired)
	mux.HandleFunc("POST /api/pairing/computers/{id}/pair", h.pairComputer)
	mux.HandleFunc("POST /api/pairing/computers/{id}/repair", retired)
	mux.HandleFunc("DELETE /api/pairing/computers/{id}", h.deleteComputer)
	mux.HandleFunc("GET /api/pairing/computers/{id}/mcp-token", h.getMCPToken)
	mux.HandleFunc("POST /api/pairing/computers/{id}/mcp-token", h.mintMCPToken)
	mux.HandleFunc("DELETE /api/pairing/computers/{id}/mcp-token", h.revokeMCPToken)
	mux.HandleFunc("GET /api/pairing/computers/{id}/projects", h.listProjects)
	mux.HandleFunc("POST /api/pairing/computers/{id}/projects", h.addProject)
	mux.HandleFunc("GET /api/pairing/computers/{id}/providers", h.listProviders)
	mux.HandleFunc("GET /api/pairing/computers/{id}/containers", h.listContainers)
	mux.HandleFunc("GET /api/pairing/computers/{id}/containers/{name}/logs", h.containerLogs)
	mux.HandleFunc("GET /api/pairing/computers/{id}/runner/logs", h.runnerLogs)
	// Read-only on purpose: a setup confirmation is written only through MCP (ADR 0063).
	mux.HandleFunc("GET /api/pairing/computers/{id}/setup", h.getSetup)
	mux.HandleFunc("PUT /api/pairing/computers/{id}/setup/choices", h.saveSetupChoices)
	mux.HandleFunc("POST /api/pairing/computers/{id}/setup/runs", h.startSetup)
	mux.HandleFunc("POST /api/pairing/computers/{id}/setup/providers/{provider}/retry", h.retrySetupProvider)
	mux.HandleFunc("POST /api/pairing/computers/{id}/setup/skills", h.updateSkills)
	mux.HandleFunc("GET /api/pairing/resolve", h.resolve)
	mux.HandleFunc("GET /api/pairing/defaults", h.getDefaults)
	mux.HandleFunc("PUT /api/pairing/defaults", h.setDefaults)
	mux.HandleFunc("GET /api/pairing/projects", h.listProjectLinks)
	mux.HandleFunc("GET /api/pairing/projects/{id}", h.getProjectLink)
	mux.HandleFunc("PUT /api/pairing/projects/{id}", h.setProjectLink)
	mux.HandleFunc("DELETE /api/pairing/projects/{id}", h.clearProjectLink)
	return mux
}

func (h *Handler) listComputers(w http.ResponseWriter, r *http.Request) {
	cs, err := h.svc.ListComputers(r.Context(), actorID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"computers": cs})
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.ListProjects(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	canClone, err := h.svc.CanClone(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"projects": projects, "can_clone": canClone})
}

// addProjectRequest names a repository to clone or, without one, the folder of an empty project.
type addProjectRequest struct {
	Repository string `json:"repository"`
	Name       string `json:"name"`
}

func (h *Handler) addProject(w http.ResponseWriter, r *http.Request) {
	var req addProjectRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	project, err := h.svc.AddProject(r.Context(), actorID(r), r.PathValue("id"), req.Repository, req.Name)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, project)
}

func (h *Handler) listProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.svc.ListProviders(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"providers": providers})
}

func (h *Handler) listContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := h.svc.ListContainers(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"containers": containers})
}

func (h *Handler) containerLogs(w http.ResponseWriter, r *http.Request) {
	containerlog.ServeSnapshot(w, r, h.svc.containerLogOpener(actorID(r), r.PathValue("id"), r.PathValue("name")), r.PathValue("name"))
}

func (h *Handler) runnerLogs(w http.ResponseWriter, r *http.Request) {
	containerlog.ServeSnapshot(w, r, h.svc.runnerLogOpener(actorID(r), r.PathValue("id")), "the runner")
}

// RunnerLogsSocket serves the live tail of a computer's runner's own log to its owner or a person its share lets read logs.
func (h *Handler) RunnerLogsSocket(w http.ResponseWriter, r *http.Request) {
	containerlog.ServeSocket(w, r, h.svc.runnerLogOpener(actorID(r), r.PathValue("id")))
}

// ContainerLogsSocket serves one container's live tail to its computer's owner or a person its share lets read logs; closing it stops docker logs there.
func (h *Handler) ContainerLogsSocket(w http.ResponseWriter, r *http.Request) {
	containerlog.ServeSocket(w, r, h.svc.containerLogOpener(actorID(r), r.PathValue("id"), r.PathValue("name")))
}

func (h *Handler) getSetup(w http.ResponseWriter, r *http.Request) {
	setup, err := h.svc.GetSetup(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, setup)
}

// startSetupRequest's Models maps a provider's driver kind to the model its setup turn runs on, ModelOptions to that
// model's options; Folder picks where the turns run; Providers lists the driver kinds to set up, empty for every provider the harness lists.
type startSetupRequest struct {
	Models       map[string]string                  `json:"models"`
	ModelOptions map[string][]harness.OptionSetting `json:"model_options"`
	Folder       string                             `json:"folder"`
	Providers    []string                           `json:"providers"`
}

func (h *Handler) startSetup(w http.ResponseWriter, r *http.Request) {
	var req startSetupRequest
	if err := optionalJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	run, err := h.svc.StartSetup(r.Context(), actorID(r), r.PathValue("id"), req.Models, req.ModelOptions, req.Folder, req.Providers)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, run)
}

func (h *Handler) saveSetupChoices(w http.ResponseWriter, r *http.Request) {
	var req SetupChoices
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	setup, err := h.svc.SaveSetupChoices(r.Context(), actorID(r), r.PathValue("id"), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, setup)
}

type retrySetupRequest struct {
	Model        string                  `json:"model"`
	ModelOptions []harness.OptionSetting `json:"model_options"`
	Folder       string                  `json:"folder"`
}

func (h *Handler) retrySetupProvider(w http.ResponseWriter, r *http.Request) {
	var req retrySetupRequest
	if err := optionalJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	run, err := h.svc.RetrySetupProvider(r.Context(), actorID(r), r.PathValue("id"), r.PathValue("provider"), req.Model, req.ModelOptions, req.Folder)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, run)
}

func (h *Handler) updateSkills(w http.ResponseWriter, r *http.Request) {
	run, err := h.svc.UpdateSkills(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, run)
}

// optionalJSON decodes the body when there is one; a bodiless POST keeps every field at its default.
func optionalJSON(r *http.Request, v any) error {
	if r.ContentLength == 0 {
		return nil
	}
	return httpx.DecodeJSON(r, v)
}

// retired answers a route of pairing through a tunnel or by URL, kept mounted because the API only grows (ADR 0082).
func retired(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusGone, httpx.Envelope{Message: "Pairing through a tunnel was retired; add the computer with the Nexul app", Code: "GONE"})
}

// pairComputer pairs an existing computer now, through its runner.
func (h *Handler) pairComputer(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.PairComputer(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		writePairError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

// writePairError keys a failure under the input that caused it, so the pairing form shows it on that field.
func writePairError(w http.ResponseWriter, err error) {
	var fe *FieldError
	if errors.As(err, &fe) {
		httpx.WriteFieldError(w, err, fe.Field)
		return
	}
	httpx.WriteError(w, err)
}

// enrollComputerRequest adds a computer, or with id mints a fresh command for one of the caller's computers.
type enrollComputerRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *Handler) enrollComputer(w http.ResponseWriter, r *http.Request) {
	var req enrollComputerRequest
	if err := optionalJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	enrolled, err := h.enrollment(r, req)
	if err != nil {
		writePairError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, enrolled)
}

func (h *Handler) enrollment(r *http.Request, req enrollComputerRequest) (*ComputerEnrollment, error) {
	if req.ID != "" {
		return h.svc.EnrollComputer(r.Context(), actorID(r), req.ID)
	}
	return h.svc.AddComputer(r.Context(), actorID(r), req.Name)
}

// updateComputer patches a computer's name, runs at once and chat folder; an omitted field keeps its value.
func (h *Handler) updateComputer(w http.ResponseWriter, r *http.Request) {
	var req updateComputerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.UpdateComputer(r.Context(), actorID(r), r.PathValue("id"), ComputerPatch(req))
	if err != nil {
		writePairError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

type updateComputerRequest struct {
	Name          *string `json:"name"`
	RunsAtOnce    *int    `json:"runs_at_once"`
	ChatProjectID *string `json:"chat_project_id"`
}

type reorderRequest struct {
	ComputerIDs []string `json:"computer_ids"`
}

func (h *Handler) reorderComputers(w http.ResponseWriter, r *http.Request) {
	var req reorderRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.ReorderComputers(r.Context(), actorID(r), req.ComputerIDs); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]string{"computer_ids": req.ComputerIDs})
}

func (h *Handler) deleteComputer(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteComputer(r.Context(), actorID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) getMCPToken(w http.ResponseWriter, r *http.Request) {
	token, err := h.svc.GetMCPToken(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"mcp_token": token})
}

// mintMCPToken is the one response that carries the raw token; nothing lists it again.
func (h *Handler) mintMCPToken(w http.ResponseWriter, r *http.Request) {
	minted, err := h.svc.MintMCPToken(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, minted)
}

func (h *Handler) revokeMCPToken(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RevokeMCPToken(r.Context(), actorID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// resolve answers whether the caller can run a play right now, without starting a turn or leaking the bearer token.
func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	target, err := h.svc.PreviewTarget(r.Context(), actorID(r), r.URL.Query().Get("project_id"))
	if err != nil {
		var nc *NotConfiguredError
		if errors.As(err, &nc) {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": false, "reason": nc.Reason, "passed_over": nc.PassedOver})
			return
		}
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":                 true,
		"computer_id":        target.Computer.ID,
		"harness_project_id": target.HarnessProjectID,
		"provider":           target.Provider,
		"model":              target.Model,
		"model_options":      target.ModelOptions,
		"passed_over":        target.PassedOver,
	})
}

func (h *Handler) getDefaults(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.GetDefaults(r.Context(), actorID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

type defaultsRequest struct {
	Provider           string                  `json:"provider"`
	Model              string                  `json:"model"`
	ModelOptions       []harness.OptionSetting `json:"model_options"`
	StartIn            StartIn                 `json:"start_in"`
	Placement          Placement               `json:"placement"`
	CloneAutomatically bool                    `json:"clone_automatically"`
}

func (h *Handler) setDefaults(w http.ResponseWriter, r *http.Request) {
	var req defaultsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	d, err := h.svc.SetDefaults(r.Context(), actorID(r), Defaults{
		Provider: req.Provider, Model: req.Model, ModelOptions: req.ModelOptions, StartIn: req.StartIn,
		Placement: req.Placement, CloneAutomatically: req.CloneAutomatically,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) listProjectLinks(w http.ResponseWriter, r *http.Request) {
	links, err := h.svc.ListProjectLinks(r.Context(), actorID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"links": links})
}

func (h *Handler) getProjectLink(w http.ResponseWriter, r *http.Request) {
	link, err := h.svc.GetProjectLink(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, link)
}

type projectLinkRequest struct {
	Provider     string                  `json:"provider"`
	Model        string                  `json:"model"`
	ModelOptions []harness.OptionSetting `json:"model_options"`
	StartIn      StartIn                 `json:"start_in"`
	Placement    Placement               `json:"placement"`
	Computers    []string                `json:"computers"`
	Checkouts    []Checkout              `json:"checkouts"`
}

func (h *Handler) setProjectLink(w http.ResponseWriter, r *http.Request) {
	var req projectLinkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	link, err := h.svc.SetProjectLink(r.Context(), actorID(r), r.PathValue("id"), ProjectLink{
		Provider: req.Provider, Model: req.Model, ModelOptions: req.ModelOptions, StartIn: req.StartIn,
		Placement: req.Placement, Computers: req.Computers, Checkouts: req.Checkouts,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, link)
}

func (h *Handler) clearProjectLink(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ClearProjectLink(r.Context(), actorID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// actorID reads the identity injected by the composition root's middleware; pairing never imports auth (ADR 0017).
func actorID(r *http.Request) string {
	if a, ok := identity.ActorFromCtx(r.Context()); ok {
		return a.ID
	}
	return ""
}
