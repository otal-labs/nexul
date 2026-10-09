import { getDeployKey, getDeployLogKey } from "@/hooks/DeployHooks";
import { getDnsExposuresKey, getDnsGatewaysKey, getDnsTunnelsKey, getDnsZonesKey } from "@/hooks/DnsHooks";
import { getDocClarificationKey, getDocKey, getDocsKey, getDocWatchersKey } from "@/hooks/DocHooks";
import { getDocFoldersKey } from "@/hooks/DocFolderHooks";
import { getInstanceUpgradeKey } from "@/hooks/InstanceUpgradeHooks";
import { getProjectPeopleKey, getWorkspacePeopleKey } from "@/hooks/PeopleHooks";
import { getProjectAccessKey } from "@/hooks/ProjectHooks";
import { getWorkspacesKey } from "@/hooks/WorkspaceHooks";
import { getTeamKey } from "@/models/Team";
import { getInterviewAnswersKey, getMemoriesKey, getMemoryKey, getMemoryVersionsKey } from "@/hooks/MemoryHooks";
import { getInterviewDraftsKey, getInterviewSourcesKey } from "@/hooks/InterviewSourceHooks";
import { getNotificationsKey, getUnreadCountKey } from "@/hooks/NotificationHooks";
import { getMeKey, getPATsKey, getSessionsKey } from "@/hooks/AuthHooks";
import { getBotwebhooksKey } from "@/hooks/BotwebhookHooks";
import { getComputerSetupKey } from "@/hooks/ComputerSetupHooks";
import { getComputersKey, getHarnessProvidersKey, getHarnessResolveKey, getMCPTokenKey } from "@/hooks/PairingHooks";
import { getWorkspaceRolesKey } from "@/hooks/RoleHooks";
import { getRunnerQueueKey, getRunnersKey } from "@/hooks/RunnerHooks";
import { getServiceDeploysKey, getServicesKey } from "@/hooks/ServiceHooks";
import { getStackDeploysKey } from "@/hooks/StackHooks";
import { getCategoriesKey, getProjectCategoriesKey } from "@/hooks/CategoryHooks";
import { getProjectTicketTypesKey, getTicketTypesKey } from "@/hooks/TicketTypeHooks";
import { getProjectStatusesKey, getStatusesKey } from "@/hooks/StatusHooks";
import { getTicketsKey } from "@/hooks/TicketCache";
import { getBlockersKey, getTicketLinkSetKey } from "@/hooks/TicketLinkHooks";
import { getChatConversationsKey, getChatUnreadKey } from "@/hooks/ChatHooks";
import { getServerVersionKey } from "@/hooks/VersionHooks";
import { getApplicablePlaysKey, getWorkspacePlaysKey } from "@/hooks/PlayHooks";
import { TEMPLATE_QUERY_KEYS } from "@/hooks/TemplateHooks";

// The queries each pushed topic refetches; frames the browser applies itself are in liveFrameHandlers.
export const pushTopics: Record<string, string[]> = {
  "runner.connected": [getRunnersKey],
  "runner.disconnected": [getRunnersKey],
  "deploy.build_started": [getRunnersKey, getRunnerQueueKey],
  "deploy.build_progress": [getRunnersKey],
  "deploy.build_completed": [getRunnersKey, getRunnerQueueKey],
  "deploy.deploy_progress": [getRunnersKey],
  "deploy.status_changed": [getRunnersKey, getRunnerQueueKey],
  // Deploy reads refetch only on deploy.updated: the runner topics above fire before the deploy domain has
  // committed, so a refetch on them can read the record from before the change.
  // ponytail: whole-log refetch per batch (≤ 4/s); append lines into the cache if logs get large.
  "deploy.updated": [getDeployKey, getDeployLogKey, getStackDeploysKey, getServiceDeploysKey],
  "service.created": [getServicesKey],
  "service.updated": [getServicesKey],
  "service.deleted": [getServicesKey],
  "dns.record_changed": [getDnsZonesKey],
  "dns.tunnel_changed": [getDnsTunnelsKey],
  "dns.gateway_changed": [getDnsGatewaysKey],
  "dns.exposure_changed": [getDnsExposuresKey],
  "notification.created": [getNotificationsKey, getUnreadCountKey],
  // The pickers' "needs setup" tags follow a setup turn confirming or withdrawing a provider.
  "computer.setup_confirmed": [getHarnessProvidersKey, getComputerSetupKey],
  "computer.setup_unconfirmed": [getHarnessProvidersKey, getComputerSetupKey],
  // The Set up step's rows and each computer row's provider lines follow a run turn by turn.
  "computer.setup_turn_changed": [getComputerSetupKey],
  "computer.setup_finished": [getComputerSetupKey, getHarnessProvidersKey],
  // A computer row goes from pairing in progress to paired, or appears and leaves, without a refresh.
  "computer.paired": [getComputersKey, getHarnessResolveKey],
  // A computer whose T3 Code moved to its new orchestrator shows its new version without a refresh.
  "computer.harness_switched": [getComputersKey, getHarnessResolveKey],
  "computer.tunnel_created": [getComputersKey],
  "computer.tunnel_removed": [getComputersKey, getHarnessResolveKey],
  // A computer row's MCP token line follows a mint or revoke from setup, the row, or an MCP tool.
  "personal_access_token.minted": [getMCPTokenKey, getPATsKey],
  "personal_access_token.revoked": [getMCPTokenKey, getPATsKey],
  // The Devices list follows a phone connecting or a device being signed out, without a refresh.
  "session.created": [getSessionsKey],
  "session.revoked": [getSessionsKey],
  // An open Bots section follows a bot made, changed, deleted, or restored anywhere, its URL included.
  "botwebhook.created": [getBotwebhooksKey],
  "botwebhook.updated": [getBotwebhooksKey],
  "botwebhook.deleted": [getBotwebhooksKey],
  "botwebhook.restored": [getBotwebhooksKey],
  "category.created": [getCategoriesKey, getProjectCategoriesKey],
  "category.updated": [getCategoriesKey, getProjectCategoriesKey],
  "category.deleted": [getCategoriesKey, getProjectCategoriesKey],
  "doc.created": [getDocsKey],
  "doc.watchers.changed": [getDocWatchersKey],
  "doc.clarification.round_started": [getDocClarificationKey],
  "doc.clarification.round_posted": [getDocClarificationKey],
  "doc.clarification.round_ended": [getDocClarificationKey],
  "doc.clarification.round_answered": [getDocClarificationKey],
  "doc.clarification.answer_saved": [getDocClarificationKey],
  "doc.clarification.answer_cleared": [getDocClarificationKey],
  "doc.clarification.anything_else_saved": [getDocClarificationKey],
  "doc.clarification.closed": [getDocClarificationKey],
  // The inbox groups doc rows by the folder each doc is in now, so a move, rename, or delete regroups it.
  "doc.moved": [getDocsKey, getDocKey, getDocFoldersKey, getNotificationsKey],
  "doc.folder.created": [getDocFoldersKey],
  "doc.folder.updated": [getDocFoldersKey, getNotificationsKey],
  "doc.folder.deleted": [getDocFoldersKey, getDocsKey, getNotificationsKey],
  "ticket.link_created": [getTicketLinkSetKey, getBlockersKey],
  "ticket.link_deleted": [getTicketLinkSetKey, getBlockersKey],
  "ticket_type.created": [getTicketTypesKey, getProjectTicketTypesKey],
  "ticket_type.updated": [getTicketTypesKey, getProjectTicketTypesKey],
  "ticket_type.deleted": [getTicketTypesKey, getProjectTicketTypesKey],
  "status.created": [getStatusesKey, getProjectStatusesKey, getTicketsKey],
  "status.updated": [getStatusesKey, getProjectStatusesKey, getTicketsKey, getTicketLinkSetKey, getBlockersKey],
  "status.deleted": [getStatusesKey, getProjectStatusesKey, getTicketsKey],
  "chat.conversation.created": [getChatConversationsKey],
  "chat.conversation.updated": [getChatConversationsKey],
  "chat.conversation.deleted": [getChatConversationsKey, getChatUnreadKey],
  // A channel switched private or public, or someone added or removed, appears in or drops from each reader's list.
  "chat.conversation.members_changed": [getChatConversationsKey, getChatUnreadKey],
  "chat.message.created": [getChatConversationsKey, getChatUnreadKey],
  "chat.message.deleted": [getChatUnreadKey],
  "instance.upgrade_changed": [getInstanceUpgradeKey, getServerVersionKey],
  "play.created": [getWorkspacePlaysKey, getApplicablePlaysKey],
  "play.updated": [getWorkspacePlaysKey, getApplicablePlaysKey],
  "play.deleted": [getWorkspacePlaysKey, getApplicablePlaysKey],
  // An instance template changes what every unedited workspace shows and what each copy is compared with.
  "instance_template.updated": TEMPLATE_QUERY_KEYS,
  "memory.created": [getMemoriesKey],
  "memory.updated": [getMemoriesKey, getMemoryKey, getMemoryVersionsKey, getInterviewSourcesKey],
  "memory.deleted": [getMemoriesKey, getMemoryKey, getInterviewSourcesKey],
  "interview_answer.saved": [getInterviewAnswersKey],
  "interview_answer.cleared": [getInterviewAnswersKey],
  "interview_source.added": [getInterviewSourcesKey],
  "interview_source.changed": [getInterviewSourcesKey],
  "interview_source.removed": [getInterviewSourcesKey],
  "interview_draft.saved": [getInterviewDraftsKey],
  "interview_draft.dismissed": [getInterviewDraftsKey],
  // The Team list and a person's detail follow account and membership changes made anywhere, MCP included.
  "account.admitted": [getTeamKey],
  "account.disabled": [getTeamKey],
  "account.reactivated": [getTeamKey],
  "account.removed": [getTeamKey, getWorkspacePeopleKey],
  "account.restored": [getTeamKey],
  // Someone's first socket opening or last one closing; the frame names nobody, the refetch applies the Team's scoping.
  "account.presence_changed": [getTeamKey],
  // A new name or picture reaches every open screen that shows the person, the saver's other devices included.
  "account.profile_updated": [getWorkspacePeopleKey, getTeamKey, getMeKey],
  "workspace.member.added": [getTeamKey, getWorkspacePeopleKey, getWorkspacesKey],
  "workspace.member.removed": [getTeamKey, getWorkspacePeopleKey, getWorkspacesKey],
  "workspace.member.updated": [getTeamKey, getProjectAccessKey, getProjectPeopleKey],
  // Someone else's Project access moved: who a manager sees with access, and who the pickers offer.
  "access.grant.changed": [getTeamKey, getProjectAccessKey, getProjectPeopleKey],
  "role.updated": [getWorkspaceRolesKey],
};
