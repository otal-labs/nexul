package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// Internal bus topics the runner domain feeds from Docker's event stream; consumed here, never part of the catalog.
const (
	TopicRunnerContainerEvent  = "runner.container_event"
	TopicRunnerContainerStates = "runner.container_states"
)

// Docker container actions a machine runner forwards.
const (
	containerDie   = "die"
	containerOOM   = "oom"
	containerStart = "start"
	containerKill  = "kill"
	containerStop  = "stop"
)

// Container states a runner's snapshot reports; every other state is left out of it.
const (
	snapshotRunning = "running"
	snapshotExited  = "exited"
)

// outageSteady is how long a restarted container must stay up before its outage is over.
const outageSteady = 60 * time.Second

// Windows in which a die is read against the event before it: Docker emits kill (and stop) for an API-requested stop
// and oom for a memory kill ahead of the die, and the kernel's own OOM kill emits no kill.
const (
	oomWindow  = 10 * time.Second
	stopWindow = 30 * time.Second
)

// outageCloseTimeout bounds the write a recovery timer makes outside any request.
const outageCloseTimeout = 10 * time.Second

// Outage is one stretch of a service being down: from its container dying non-zero or being OOM-killed until it has
// stayed running; a restart loop is one outage whose RestartCount grows.
type Outage struct {
	ID           string     `json:"id"`
	StackID      string     `json:"stack_id"`
	Service      string     `json:"service"`
	Container    string     `json:"container"`
	StartedAt    time.Time  `json:"started_at"`
	ExitCode     int        `json:"exit_code"`
	OOM          bool       `json:"oom"`
	RestartCount int        `json:"restart_count"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
}

// OutageEvent is the service.container_stopped and service.container_recovered payload. It carries no output lines:
// event payloads are stored, and container output is read live (ADR 0091).
type OutageEvent struct {
	WorkspaceID  string     `json:"workspace_id,omitempty" jsonschema:"The workspace of the stack's project; empty for an instance stack."`
	ProjectID    string     `json:"project_id,omitempty" jsonschema:"The stack's project; empty for an instance stack."`
	StackID      string     `json:"stack_id" jsonschema:"The stack the service belongs to."`
	StackName    string     `json:"stack_name" jsonschema:"The stack's name."`
	Service      string     `json:"service" jsonschema:"The service's name in the stack; read its output with stack_get and logs for this service."`
	Container    string     `json:"container" jsonschema:"The container's name on the machine."`
	Machine      string     `json:"machine" jsonschema:"The machine the stack runs on."`
	OutageID     string     `json:"outage_id" jsonschema:"The outage this event opens or closes."`
	ExitCode     int        `json:"exit_code" jsonschema:"The exit code of the last crash; 0 when only an out-of-memory kill was seen so far."`
	OOM          bool       `json:"oom" jsonschema:"True when the kernel killed the container for running out of memory."`
	RestartCount int        `json:"restart_count" jsonschema:"How many more times the container crashed after the first."`
	StartedAt    time.Time  `json:"started_at" jsonschema:"When the service went down."`
	EndedAt      *time.Time `json:"ended_at,omitempty" jsonschema:"When the service was steady again; sent only with service.container_recovered."`
}

// containerEvent mirrors the runner's container event independently (ADR 0017).
type containerEvent struct {
	Machine   string `json:"machine"`
	Container string `json:"container"`
	Project   string `json:"project,omitempty"`
	Service   string `json:"service,omitempty"`
	Action    string `json:"action"`
	ExitCode  int    `json:"exit_code,omitempty"`
}

// containerStates mirrors the runner's snapshot of a machine's containers.
type containerStates struct {
	Machine string           `json:"machine"`
	States  []containerState `json:"states"`
}

type containerState struct {
	Container  string `json:"container"`
	Project    string `json:"project,omitempty"`
	Service    string `json:"service,omitempty"`
	State      string `json:"state"`
	ExitCode   int    `json:"exit_code,omitempty"`
	OOM        bool   `json:"oom,omitempty"`
	FinishedAt int64  `json:"finished_at,omitempty"`
}

// recovery is one pending steady-state timer; a fired timer that is no longer the registered one was superseded.
type recovery struct{ timer *time.Timer }

// SetOutageRepo wires outage storage; unset, container events are ignored and no stack has outages.
func (s *Service) SetOutageRepo(r OutageRepo) {
	s.outages = r
	s.recoveries = map[string]*recovery{}
	s.containerMarks = map[string]time.Time{}
}

// OpenOutages lists the stack's services that are down. The caller has already read the stack.
func (s *Service) OpenOutages(ctx context.Context, stackID string) ([]*Outage, error) {
	if s.outages == nil {
		return nil, nil
	}
	open, err := s.outages.ListOpenByStack(ctx, stackID)
	if err != nil {
		return nil, fmt.Errorf("list open outages of stack %s: %w", stackID, err)
	}
	return open, nil
}

// HandleContainerEvent is the runner.container_event consumer: a death opens the service's outage unless a kill or stop
// asked for it, a start begins the steady-state wait that ends it. An OOM kill only marks the container, since the
// kernel may have killed a child and left the container running; the death that follows is the outage.
func (s *Service) HandleContainerEvent(ctx context.Context, ev eventbus.Event) error {
	var p containerEvent
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", TopicRunnerContainerEvent, err))
	}
	if s.outages == nil {
		return nil
	}
	s.outageMu.Lock()
	defer s.outageMu.Unlock()
	stack, service, err := s.outageTarget(ctx, p.Machine, p.Project, p.Container, p.Service)
	if err != nil || stack == nil {
		return err
	}
	switch p.Action {
	case containerStart:
		delete(s.containerMarks, markKey(containerKill, stack.ID, p.Container))
		delete(s.containerMarks, markKey(containerStop, stack.ID, p.Container))
		return s.containerStarted(ctx, stack, service)
	case containerOOM, containerKill, containerStop:
		s.markContainer(p.Action, stack.ID, p.Container)
	case containerDie:
		return s.containerStopped(ctx, stack, service, p)
	}
	return nil
}

func markKey(kind, stackID, container string) string { return kind + "/" + stackID + "/" + container }

// markContainer remembers an event that colours the die after it, dropping marks too old for any die to read.
func (s *Service) markContainer(kind, stackID, container string) {
	now := s.now()
	for k, at := range s.containerMarks {
		if now.Sub(at) > stopWindow {
			delete(s.containerMarks, k)
		}
	}
	s.containerMarks[markKey(kind, stackID, container)] = now
}

// marked consumes the mark of a kind on a container and says whether it is still within window.
func (s *Service) marked(kind, stackID, container string, window time.Duration) bool {
	key := markKey(kind, stackID, container)
	at, ok := s.containerMarks[key]
	delete(s.containerMarks, key)
	return ok && s.now().Sub(at) <= window
}

// HandleContainerStates is the runner.container_states consumer: a runner's report of what its containers are now
// settles what the event stream missed while it was disconnected, and changes nothing the stream already settled.
func (s *Service) HandleContainerStates(ctx context.Context, ev eventbus.Event) error {
	var p containerStates
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", TopicRunnerContainerStates, err))
	}
	if s.outages == nil {
		return nil
	}
	s.outageMu.Lock()
	defer s.outageMu.Unlock()
	for _, st := range p.States {
		stack, service, err := s.outageTarget(ctx, p.Machine, st.Project, st.Container, st.Service)
		if err != nil {
			return err
		}
		if stack == nil {
			continue
		}
		if err := s.settleState(ctx, stack, service, st); err != nil {
			return err
		}
	}
	return nil
}

// outageTarget finds the stack and service a container belongs to on a machine: by compose project, else by the
// container's own name for a run stack, and names the service as the stack's own rows do. It answers nil for a container Nexul did not deploy and for a stack with a
// deploy under way, whose recreated containers are not crashes.
func (s *Service) outageTarget(ctx context.Context, machine, project, container, service string) (*Stack, string, error) {
	if machine == "" {
		return nil, "", nil
	}
	slug := project
	if slug == "" {
		slug = container
	}
	stack, err := s.stacks.GetBySlugAndMachine(ctx, slug, machine)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("find stack %s on %s: %w", slug, machine, err)
	}
	busy, err := s.repo.HasActive(ctx, stack.ID)
	if err != nil {
		return nil, "", fmt.Errorf("check deploys of stack %s: %w", stack.ID, err)
	}
	if busy {
		return nil, "", nil
	}
	service, err = s.serviceName(ctx, stack, container, service)
	if err != nil {
		return nil, "", err
	}
	return stack, service, nil
}

// serviceName is the name the stack's service rows give a container: an imported compose stack names its rows after
// the containers, a deployed one after the compose service, so the compose label is only the answer for a container
// no row holds yet.
func (s *Service) serviceName(ctx context.Context, stack *Stack, container, label string) (string, error) {
	rows, err := s.services.ListByStack(ctx, stack.ID)
	if err != nil {
		return "", fmt.Errorf("list services of stack %s: %w", stack.ID, err)
	}
	if i := slices.IndexFunc(rows, func(c *Container) bool { return c.ContainerName == container }); i >= 0 {
		return rows[i].Name, nil
	}
	if label == "" {
		return stack.Slug, nil
	}
	return label, nil
}

func (s *Service) containerStopped(ctx context.Context, stack *Stack, service string, p containerEvent) error {
	s.cancelRecovery(stack.ID, service)
	oom := s.marked(containerOOM, stack.ID, p.Container, oomWindow)
	asked := s.marked(containerKill, stack.ID, p.Container, stopWindow)
	asked = s.marked(containerStop, stack.ID, p.Container, stopWindow) || asked
	if asked || p.ExitCode == 0 {
		return nil
	}
	open, err := s.outages.GetOpen(ctx, stack.ID, service)
	if err == nil {
		return s.continueOutage(ctx, open, p, oom)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("get open outage of %s: %w", service, err)
	}
	return s.openOutage(ctx, stack, &Outage{
		ID: ids.New(), StackID: stack.ID, Service: service, Container: p.Container,
		StartedAt: s.now().UTC(), ExitCode: p.ExitCode, OOM: oom,
	})
}

// continueOutage records another crash on an open outage without announcing it again.
func (s *Service) continueOutage(ctx context.Context, open *Outage, p containerEvent, oom bool) error {
	open.Container = p.Container
	open.OOM = open.OOM || oom
	open.RestartCount++
	open.ExitCode = p.ExitCode
	if err := s.outages.Update(ctx, open); err != nil {
		return fmt.Errorf("update outage %s: %w", open.ID, err)
	}
	return nil
}

func (s *Service) openOutage(ctx context.Context, stack *Stack, o *Outage) error {
	evt, err := s.outageEvent(ctx, TopicContainerStopped, stack, o)
	if err != nil {
		return err
	}
	if err := s.outages.Open(ctx, o, evt); err != nil {
		return fmt.Errorf("open outage of %s: %w", o.Service, err)
	}
	return nil
}

func (s *Service) containerStarted(ctx context.Context, stack *Stack, service string) error {
	open, err := s.outages.GetOpen(ctx, stack.ID, service)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get open outage of %s: %w", service, err)
	}
	s.armRecovery(stack, open)
	return nil
}

// settleState applies one reported container to its service: running ends an open outage, an exit that no outage
// holds yet opens one unless the service has been up since it happened.
func (s *Service) settleState(ctx context.Context, stack *Stack, service string, st containerState) error {
	open, err := s.outages.GetOpen(ctx, stack.ID, service)
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("get open outage of %s: %w", service, err)
	}
	if st.State == snapshotRunning && open != nil {
		s.cancelRecovery(stack.ID, service)
		return s.closeOutage(ctx, stack, open)
	}
	if st.State != snapshotExited || open != nil || (st.ExitCode == 0 && !st.OOM) {
		return nil
	}
	finished := time.Unix(st.FinishedAt, 0).UTC()
	if st.FinishedAt <= 0 {
		finished = s.now().UTC()
	}
	last, err := s.outages.LatestEnd(ctx, stack.ID, service)
	if err != nil {
		return fmt.Errorf("get last outage end of %s: %w", service, err)
	}
	if !finished.After(last) {
		return nil
	}
	return s.openOutage(ctx, stack, &Outage{
		ID: ids.New(), StackID: stack.ID, Service: service, Container: st.Container,
		StartedAt: finished, ExitCode: st.ExitCode, OOM: st.OOM,
	})
}

func (s *Service) closeOutage(ctx context.Context, stack *Stack, open *Outage) error {
	end := s.now().UTC()
	open.EndedAt = &end
	evt, err := s.outageEvent(ctx, TopicContainerRecovered, stack, open)
	if err != nil {
		return err
	}
	if err := s.outages.Close(ctx, open.ID, end, evt); err != nil {
		return fmt.Errorf("close outage %s: %w", open.ID, err)
	}
	return nil
}

func recoveryKey(stackID, service string) string { return stackID + "/" + service }

// armRecovery starts the steady-state wait for an open outage, replacing one already running.
func (s *Service) armRecovery(stack *Stack, open *Outage) {
	s.cancelRecovery(stack.ID, open.Service)
	key, r := recoveryKey(stack.ID, open.Service), &recovery{}
	r.timer = time.AfterFunc(outageSteady, func() { s.recoverSteady(key, r, stack, open.Service) })
	s.recoveries[key] = r
}

// cancelRecovery drops the pending wait of a service whose container died or whose outage ended another way.
func (s *Service) cancelRecovery(stackID, service string) {
	key := recoveryKey(stackID, service)
	r, ok := s.recoveries[key]
	if !ok {
		return
	}
	r.timer.Stop()
	delete(s.recoveries, key)
}

// recoverSteady ends the outage of a container that stayed up for the whole wait.
func (s *Service) recoverSteady(key string, r *recovery, stack *Stack, service string) {
	s.outageMu.Lock()
	defer s.outageMu.Unlock()
	if s.recoveries[key] != r {
		return
	}
	delete(s.recoveries, key)
	ctx, cancel := context.WithTimeout(context.Background(), outageCloseTimeout)
	defer cancel()
	open, err := s.outages.GetOpen(ctx, stack.ID, service)
	if errors.Is(err, apperrs.ErrNotFound) {
		return
	}
	if err == nil {
		err = s.closeOutage(ctx, stack, open)
	}
	if err != nil {
		slog.Warn("end outage after steady wait failed", "stack_id", stack.ID, "service", service, "error", err)
	}
}

// recoverAfterDeploy starts the steady-state wait for every open outage of a stack whose deploy came up healthy: the
// deploy recreated the containers while their events were held back, so no start would ever begin it.
func (s *Service) recoverAfterDeploy(ctx context.Context, stackID string, observed []ObservedService) error {
	s.outageMu.Lock()
	defer s.outageMu.Unlock()
	open, err := s.outages.ListOpenByStack(ctx, stackID)
	if err != nil || len(open) == 0 {
		return err
	}
	stack, err := s.stacks.GetByID(ctx, stackID)
	if err != nil {
		return fmt.Errorf("get stack %s: %w", stackID, err)
	}
	for _, o := range open {
		for _, svc := range observed {
			if svc.Name == o.Service && (svc.Status == string(ServiceStatusRunning) || svc.Status == string(ServiceStatusHealthy)) {
				s.armRecovery(stack, o)
			}
		}
	}
	return nil
}

// outageEvent builds the outbox row announcing o, written in the same transaction as the outage itself.
func (s *Service) outageEvent(ctx context.Context, topic string, stack *Stack, o *Outage) (eventbus.OutboxEvent, error) {
	var workspaceID string
	if stack.ProjectID != "" && s.projects != nil {
		id, err := s.projects.WorkspaceOf(ctx, stack.ProjectID)
		if err != nil {
			return eventbus.OutboxEvent{}, fmt.Errorf("find workspace of project %s: %w", stack.ProjectID, err)
		}
		workspaceID = id
	}
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: OutageEvent{
		WorkspaceID: workspaceID, ProjectID: stack.ProjectID, StackID: stack.ID, StackName: stack.Name,
		Service: o.Service, Container: o.Container, Machine: stack.Machine, OutageID: o.ID,
		ExitCode: o.ExitCode, OOM: o.OOM, RestartCount: o.RestartCount, StartedAt: o.StartedAt, EndedAt: o.EndedAt,
	}}, nil
}
