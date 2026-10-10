package repository

import (
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

const (
	// TopicInstallationAssigned is published when attaching a repository links its installation's account to a workspace.
	TopicInstallationAssigned = "repository.installation.assigned"
	// TopicInstallationUnassigned is published when the account is detached from it, by hand or because it was uninstalled.
	TopicInstallationUnassigned = "repository.installation.unassigned"
)

// Topics declares the repository domain's topics with their payloads.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicInstallationAssigned, Payload: InstallationEvent{}},
		{Name: TopicInstallationUnassigned, Payload: InstallationEvent{}},
	}
}

// InstallationEvent names the installation's account and the workspace that gained or lost it.
type InstallationEvent struct {
	AccountID    int64  `json:"account_id" jsonschema:"GitHub's numeric id of the account or organisation the App is installed on; 0 for an assignment made before ids were kept."`
	AccountLogin string `json:"account_login" jsonschema:"The account's login, lowercase, as GitHub last named it."`
	WorkspaceID  string `json:"workspace_id" jsonschema:"The workspace the account was linked to or detached from."`
	ActorID      string `json:"actor_id,omitempty" jsonschema:"The person who attached a repository from the account, or who detached it; earlier events name who assigned it or made the install link it was claimed through."`
	Uninstalled  bool   `json:"uninstalled,omitempty" jsonschema:"On repository.installation.unassigned, true when GitHub stopped listing the installation rather than someone detaching it."`
}

func installationEvent(topic string, a Assignment, actorID string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: InstallationEvent{
		AccountID: a.AccountID, AccountLogin: a.AccountLogin, WorkspaceID: a.WorkspaceID, ActorID: actorID, Uninstalled: a.Gone,
	}}
}
