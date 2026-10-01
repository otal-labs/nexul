// GENERATED FILE — do not edit by hand.
// Source of truth: internal/integrations/catalog.go (AM11).
// Regenerate: bun run generate:events (from sdk/).

export interface EventPayloads {
  "access.grant.changed": { "resource_type": "doc" | "play" | "project"; "resource_id": string; "user_id": string; "actor_id"?: string; };
  "account.admitted": { "invitation_id"?: string; "user_id"?: string; };
  "account.disabled": { "account_id": string; "actor_id"?: string; };
  "account.profile_updated": { "account_id": string; };
  "account.reactivated": { "account_id": string; "actor_id"?: string; };
  "account.removed": { "account_id": string; "actor_id"?: string; };
  "account.restored": { "account_id": string; "actor_id"?: string; };
  "category.created": { "category": Record<string, unknown>; };
  "category.deleted": { "category": Record<string, unknown>; };
  "category.updated": { "category": Record<string, unknown>; };
  "chat.conversation.created": { "conversation": Record<string, unknown>; "members_only"?: boolean; };
  "chat.conversation.deleted": { "conversation_id": string; "workspace_id": string; "kind": "channel" | "voice_channel"; "name": string; "actor_id"?: string; "private"?: boolean; "member_ids"?: string[]; "members_only"?: boolean; };
  "chat.conversation.members_changed": { "conversation_id": string; "workspace_id": string; "private": boolean; "added_user_ids": string[]; "removed_user_ids": string[]; "actor_id"?: string; "members_only"?: boolean; };
  "chat.conversation.updated": { "conversation_id": string; "workspace_id": string; "kind": "channel" | "voice_channel"; "name": string; "previous_name": string; "actor_id"?: string; "members_only"?: boolean; };
  "chat.message.created": { "message": Record<string, unknown>; "members_only"?: boolean; };
  "chat.message.deleted": { "conversation_id": string; "message_id": string; "deleted_at": string; "members_only"?: boolean; };
  "chat.message.updated": { "message": Record<string, unknown>; "members_only"?: boolean; };
  "computer.paired": { "computer_id": string; "user_id": string; "server_url": string; "harness_version"?: string; "token_expires_at": string; };
  "computer.setup_confirmed": { "computer_id": string; "user_id": string; "provider"?: string; "confirmed_at"?: string; "skills"?: string[]; };
  "computer.setup_finished": { "computer_id": string; "user_id": string; "run_id": string; "confirmed": boolean; "providers": { "provider": string; "state": "running" | "confirmed" | "failed"; "status": string; }[]; };
  "computer.setup_turn_activity": { "computer_id": string; "user_id": string; "run_id": string; "turn_id": string; "provider": string; "status": string; "call_id"?: string; "kind"?: "tool_call" | "tool_result" | "text" | "question" | "other"; "tool"?: string; "text"?: string; "at"?: string; };
  "computer.setup_turn_changed": { "computer_id": string; "user_id": string; "run_id": string; "turn_id": string; "provider": string; "provider_name": string; "model"?: string; "state": "running" | "confirmed" | "failed"; "status": string; "started_at": string; "ended_at"?: string; };
  "computer.setup_unconfirmed": { "computer_id": string; "user_id": string; "provider"?: string; "confirmed_at"?: string; "skills"?: string[]; };
  "computer.tunnel_created": { "computer_id": string; "user_id": string; "tunnel_id": string; "hostname": string; };
  "computer.tunnel_removed": { "computer_id": string; "user_id": string; "tunnel_id": string; "hostname": string; };
  "computer.tunnel_status_changed": { "computer_id": string; "user_id": string; "tunnel": string; "harness_reachable": boolean; "harness_version"?: string; };
  "deploy.build_completed": { "id": string; "status": string; "artifacts"?: string[]; "error"?: string; };
  "deploy.build_progress": { "id": string; "step": number; "total": number; "log"?: string; };
  "deploy.build_started": { "id": string; "total": number; "log"?: string; };
  "deploy.cancel_requested": { "id": string; };
  "deploy.deploy_progress": { "id": string; "phase": string; "log"?: string; };
  "deploy.log": { "id": string; "phase": "checkout" | "build" | "deploy"; "log": string; "ts": number; };
  "deploy.requested": { "id": string; "kind": "build" | "deploy"; "service"?: string; "target"?: string; "image"?: string; "env"?: Record<string, string>; "strategy"?: string; "repo"?: string; "ref"?: string; "compose_dir"?: string; "network"?: string; "ports"?: string[]; "mounts"?: string[]; "dockerfile"?: string; "compose_path"?: string; "health_check"?: Record<string, unknown>; };
  "deploy.status_changed": { "id": string; "status": "pending" | "running" | "healthy" | "failed"; "error"?: string; };
  "deploy.updated": { "id": string; "status": "pending" | "running" | "healthy" | "failed"; };
  "dns.exposure_changed": { "exposure_id": string; "gateway_id": string; "hostname"?: string; "service"?: string; "port"?: number; "action": "created" | "deleted"; };
  "dns.gateway_changed": { "gateway_id": string; "kind"?: string; "docker_network"?: string; "action": "created" | "deleted"; };
  "dns.record_changed": { "zone_id": string; "zone"?: string; "record_id"?: string; "action": "created" | "updated" | "deleted"; "type"?: string; "name"?: string; "service"?: string; };
  "dns.tunnel_changed": { "tunnel_id": string; "name"?: string; "action": "created" | "routed" | "rotated" | "deleted"; "hostname"?: string; "service"?: string; };
  "doc.created": { "doc": { "id": string; "project_id"?: string; "folder_id"?: string; "title": string; "body"?: string; "version": number; "archived"?: boolean; "locked"?: boolean; "created_at"?: string; "updated_at"?: string; }; "actor_id"?: string; "mentioned_user_ids"?: string[]; };
  "doc.deleted": { "id": string; "title": string; "project_id"?: string; };
  "doc.folder.created": { "folder": { "id": string; "project_id": string; "name": string; "is_default": boolean; "created_at"?: string; "updated_at"?: string; }; "actor_id"?: string; };
  "doc.folder.deleted": { "folder": { "id": string; "project_id": string; "name": string; "is_default": boolean; "created_at"?: string; "updated_at"?: string; }; "moved_to_folder_id": string; "actor_id"?: string; };
  "doc.folder.updated": { "folder": { "id": string; "project_id": string; "name": string; "is_default": boolean; "created_at"?: string; "updated_at"?: string; }; "previous_name": string; "actor_id"?: string; };
  "doc.moved": { "doc": { "id": string; "project_id": string; "folder_id": string; "title": string; }; "from_folder_id": string; "actor_id"?: string; };
  "doc.updated": { "doc": { "id": string; "project_id"?: string; "folder_id"?: string; "title": string; "body"?: string; "version": number; "archived"?: boolean; "locked"?: boolean; "created_at"?: string; "updated_at"?: string; }; "actor_id"?: string; "mentioned_user_ids"?: string[]; };
  "doc.watchers.changed": { "doc": { "id": string; "project_id": string; "title": string; }; "user_id": string; "watching": boolean; };
  "git.branch_deleted": { "owner": string; "repo": string; "branch": string; };
  "git.pr_closed": { "owner": string; "repo": string; "pr": Record<string, unknown>; };
  "git.pr_comment": { "owner": string; "repo": string; "pr": { "number": number; }; "comment": { "body": string; "author": string; }; };
  "git.pr_merged": { "owner": string; "repo": string; "pr": Record<string, unknown>; };
  "git.pr_opened": { "owner": string; "repo": string; "pr": Record<string, unknown>; };
  "git.pr_review_submitted": { "owner": string; "repo": string; "pr": Record<string, unknown>; };
  "git.provider_event": { "provider": string; "event_type": string; "delivery_id": string; "action"?: string; "repository"?: Record<string, unknown>; "payload"?: Record<string, unknown>; "received_at": string; };
  "git.push": { "owner": string; "repo": string; "branch": string; "sha": string; "pusher"?: string; };
  "identity.linked": { "user_id": string; "provider": string; "login": string; };
  "identity.unlinked": { "user_id": string; "provider": string; "login": string; };
  "instance.upgrade_changed": { "id": string; "from_version": string; "to_version": string; "status": "pending" | "started" | "completed" | "failed"; "error"?: string; "requested_by"?: string; "created_at"?: string; "updated_at"?: string; };
  "instance.upgrade_requested": { "id": string; "version": string; };
  "instance_template.updated": { "kind": "interview" | "mention_chip" | "play_instructions" | "ticket_body"; "key": string; "author_id": string; "updated_at"?: string; "reset": boolean; };
  "interview_template.updated": { "workspace_id": string; "author_id": string; "updated_at"?: string; };
  "invitation.created": { "invitation_id"?: string; "actor_id"?: string; };
  "invitation.deleted": { "invitation_id"?: string; "reason"?: string; };
  "invitation.redeemed": { "invitation_id"?: string; "user_id"?: string; };
  "invitation.revoked": { "invitation_id"?: string; "actor_id"?: string; };
  "memory.created": { "memory": { "id": string; "workspace_id": string; "project_id": string; "kind"?: string; "title": string; "when_to_use"?: string; "always_included"?: boolean; "updated_at"?: string; }; "author_id": string; };
  "memory.deleted": { "id": string; "workspace_id"?: string; "project_id"?: string; "title": string; "author_id": string; };
  "memory.updated": { "memory": { "id": string; "workspace_id": string; "project_id": string; "kind"?: string; "title": string; "when_to_use"?: string; "always_included"?: boolean; "updated_at"?: string; }; "author_id": string; };
  "notification.created": Record<string, unknown>;
  "notification.push_requested": { "notifications": { "id": string; "user_id": string; "workspace_id"?: string; }[]; };
  "personal_access_token.minted": { "token_id": string; "user_id": string; "name": string; "computer_id"?: string; };
  "personal_access_token.revoked": { "token_id": string; "user_id": string; "name": string; "computer_id"?: string; };
  "play.created": { "play": { "id": string; "workspace_id": string; "label": string; "type": string; "description"?: string; "instructions"?: string; "enabled"?: boolean; "show_when_stage"?: string | null; "excluded_project_ids"?: string[]; "created_by"?: string; "created_at"?: string; "updated_at"?: string; }; };
  "play.deleted": { "id": string; "label": string; "workspace_id"?: string; };
  "play.run_finished": { "trail_id": string; "play_id": string; "play_label": string; "target_type": "ticket" | "doc" | "interview"; "target_id": string; "target_title"?: string; "starter_id": string; "via": "web" | "mcp"; "workspace_id"?: string; "outcome": "done" | "failed" | "interrupted"; "last_error"?: string; "reply_message_id"?: string; };
  "play.run_started": { "trail_id": string; "play_id": string; "play_label": string; "target_type": "ticket" | "doc" | "interview"; "target_id": string; "target_title"?: string; "starter_id": string; "via": "web" | "mcp"; "workspace_id"?: string; "harness_session_id"?: string; };
  "play.run_waiting": { "trail_id": string; "play_id": string; "play_label": string; "target_type": "ticket" | "doc" | "interview"; "target_id": string; "target_title"?: string; "starter_id": string; "via": "web" | "mcp"; "workspace_id"?: string; };
  "play.updated": { "play": { "id": string; "workspace_id": string; "label": string; "type": string; "description"?: string; "instructions"?: string; "enabled"?: boolean; "show_when_stage"?: string | null; "excluded_project_ids"?: string[]; "created_by"?: string; "created_at"?: string; "updated_at"?: string; }; };
  "review.status_changed": { "id": string; "repo": string; "pr_number": number; "status": string; "reviewer"?: string; };
  "role.updated": { "role_id": string; "workspace_id": string; "actor_id"?: string; };
  "runner.connected": { "runner_id": string; "name"?: string; };
  "runner.disconnected": { "runner_id": string; "reason"?: string; };
  "runner.heartbeat": { "runner_id": string; "ts": number; };
  "service.created": { "service": Record<string, unknown>; };
  "service.deleted": { "id": string; "name": string; "project_id"?: string; };
  "service.updated": { "service": Record<string, unknown>; };
  "session.created": { "session_id": string; "user_id": string; "client": "browser" | "desktop" | "phone"; "platform"?: string; "label"?: string; };
  "session.revoked": { "session_id": string; "user_id": string; "client": "browser" | "desktop" | "phone"; "platform"?: string; "label"?: string; };
  "status.created": { "status": Record<string, unknown>; };
  "status.deleted": { "status": Record<string, unknown>; };
  "status.updated": { "status": Record<string, unknown>; };
  "ticket.assignee_changed": { "ticket": Record<string, unknown>; "from": string; "to": string; };
  "ticket.category_changed": { "ticket_id": string; "category_id": string; };
  "ticket.created": { "ticket": { "id": string; "project_id"?: string; "title": string; "body"?: string; "status": "open" | "in_progress" | "done" | "closed"; "doc_id"?: string; "assignee"?: string; "developer"?: string; "tester"?: string; "reporter"?: { "kind": "user" | "user:mcp" | "automation"; "login"?: string; "automation_id"?: string; "automation_name"?: string; }; "created_at"?: string; "updated_at"?: string; "finished_at"?: string | null; }; "mentioned_user_ids"?: string[]; };
  "ticket.deleted": { "id": string; "title": string; "project_id"?: string; };
  "ticket.developer_changed": { "ticket": Record<string, unknown>; "from": string; "to": string; };
  "ticket.finished": { "ticket": Record<string, unknown>; };
  "ticket.link_created": { "link": { "ticket_id": string; "kind": "found_in" | "blocked_by"; "target_id": string; "created_at"?: string; }; };
  "ticket.link_deleted": { "link": { "ticket_id": string; "kind": "found_in" | "blocked_by"; "target_id": string; "created_at"?: string; }; };
  "ticket.status_changed": { "ticket": Record<string, unknown>; "from": string; "to": string; "actor"?: { "kind"?: "user" | "automation" | "play" | "play:mcp"; "automation_id"?: string; "automation_name"?: string; "play_label"?: string; "trail_id"?: string; "user_id"?: string; }; "execution_id"?: string; };
  "ticket.test_failed": { "ticket": Record<string, unknown>; "tester": string; "report": string; };
  "ticket.test_passed": { "ticket": Record<string, unknown>; "tester": string; };
  "ticket.tester_changed": { "ticket": Record<string, unknown>; "from": string; "to": string; };
  "ticket.updated": { "ticket": Record<string, unknown>; "actor_id"?: string; "mentioned_user_ids"?: string[]; };
  "ticket_type.created": { "ticket_type": Record<string, unknown>; };
  "ticket_type.deleted": { "ticket_type": Record<string, unknown>; };
  "ticket_type.updated": { "ticket_type": Record<string, unknown>; };
  "topology.updated": { "environment": string; "canvas"?: Record<string, unknown>; };
  "voice.occupancy.changed": { "conversation_id": string; "occupants": { "identity": string; "name": string; }[]; "members_only"?: boolean; };
  "workspace.member.added": { "invitation_id"?: string; "user_id"?: string; "workspace_id"?: string; "actor_id"?: string; };
  "workspace.member.removed": { "user_id": string; "workspace_id": string; "actor_id"?: string; };
  "workspace.member.updated": { "user_id": string; "workspace_id": string; "actor_id"?: string; };
  "workspace.updated": { "workspace_id": string; "name": string; "slug": string; "actor_id"?: string; };
}

export type Topic = keyof EventPayloads;

export const TOPICS: Topic[] = [
  "access.grant.changed",
  "account.admitted",
  "account.disabled",
  "account.profile_updated",
  "account.reactivated",
  "account.removed",
  "account.restored",
  "category.created",
  "category.deleted",
  "category.updated",
  "chat.conversation.created",
  "chat.conversation.deleted",
  "chat.conversation.members_changed",
  "chat.conversation.updated",
  "chat.message.created",
  "chat.message.deleted",
  "chat.message.updated",
  "computer.paired",
  "computer.setup_confirmed",
  "computer.setup_finished",
  "computer.setup_turn_activity",
  "computer.setup_turn_changed",
  "computer.setup_unconfirmed",
  "computer.tunnel_created",
  "computer.tunnel_removed",
  "computer.tunnel_status_changed",
  "deploy.build_completed",
  "deploy.build_progress",
  "deploy.build_started",
  "deploy.cancel_requested",
  "deploy.deploy_progress",
  "deploy.log",
  "deploy.requested",
  "deploy.status_changed",
  "deploy.updated",
  "dns.exposure_changed",
  "dns.gateway_changed",
  "dns.record_changed",
  "dns.tunnel_changed",
  "doc.created",
  "doc.deleted",
  "doc.folder.created",
  "doc.folder.deleted",
  "doc.folder.updated",
  "doc.moved",
  "doc.updated",
  "doc.watchers.changed",
  "git.branch_deleted",
  "git.pr_closed",
  "git.pr_comment",
  "git.pr_merged",
  "git.pr_opened",
  "git.pr_review_submitted",
  "git.provider_event",
  "git.push",
  "identity.linked",
  "identity.unlinked",
  "instance.upgrade_changed",
  "instance.upgrade_requested",
  "instance_template.updated",
  "interview_template.updated",
  "invitation.created",
  "invitation.deleted",
  "invitation.redeemed",
  "invitation.revoked",
  "memory.created",
  "memory.deleted",
  "memory.updated",
  "notification.created",
  "notification.push_requested",
  "personal_access_token.minted",
  "personal_access_token.revoked",
  "play.created",
  "play.deleted",
  "play.run_finished",
  "play.run_started",
  "play.run_waiting",
  "play.updated",
  "review.status_changed",
  "role.updated",
  "runner.connected",
  "runner.disconnected",
  "runner.heartbeat",
  "service.created",
  "service.deleted",
  "service.updated",
  "session.created",
  "session.revoked",
  "status.created",
  "status.deleted",
  "status.updated",
  "ticket.assignee_changed",
  "ticket.category_changed",
  "ticket.created",
  "ticket.deleted",
  "ticket.developer_changed",
  "ticket.finished",
  "ticket.link_created",
  "ticket.link_deleted",
  "ticket.status_changed",
  "ticket.test_failed",
  "ticket.test_passed",
  "ticket.tester_changed",
  "ticket.updated",
  "ticket_type.created",
  "ticket_type.deleted",
  "ticket_type.updated",
  "topology.updated",
  "voice.occupancy.changed",
  "workspace.member.added",
  "workspace.member.removed",
  "workspace.member.updated",
  "workspace.updated",
];

export const eventFixtures: { [K in Topic]: EventPayloads[K] } = {
  "access.grant.changed": {"resource_type":"doc","resource_id":"fixture-resource_id","user_id":"fixture-user_id","actor_id":"fixture-actor_id"},
  "account.admitted": {"invitation_id":"fixture-invitation_id","user_id":"fixture-user_id"},
  "account.disabled": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "account.profile_updated": {"account_id":"fixture-account_id"},
  "account.reactivated": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "account.removed": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "account.restored": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "category.created": {"category":{}},
  "category.deleted": {"category":{}},
  "category.updated": {"category":{}},
  "chat.conversation.created": {"conversation":{},"members_only":false},
  "chat.conversation.deleted": {"conversation_id":"fixture-conversation_id","workspace_id":"fixture-workspace_id","kind":"channel","name":"fixture-name","actor_id":"fixture-actor_id","private":false,"member_ids":["fixture-member_ids"],"members_only":false},
  "chat.conversation.members_changed": {"conversation_id":"fixture-conversation_id","workspace_id":"fixture-workspace_id","private":false,"added_user_ids":["fixture-added_user_ids"],"removed_user_ids":["fixture-removed_user_ids"],"actor_id":"fixture-actor_id","members_only":false},
  "chat.conversation.updated": {"conversation_id":"fixture-conversation_id","workspace_id":"fixture-workspace_id","kind":"channel","name":"fixture-name","previous_name":"fixture-previous_name","actor_id":"fixture-actor_id","members_only":false},
  "chat.message.created": {"message":{},"members_only":false},
  "chat.message.deleted": {"conversation_id":"fixture-conversation_id","message_id":"fixture-message_id","deleted_at":"2026-01-01T00:00:00Z","members_only":false},
  "chat.message.updated": {"message":{},"members_only":false},
  "computer.paired": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","server_url":"fixture-server_url","harness_version":"fixture-harness_version","token_expires_at":"2026-01-01T00:00:00Z"},
  "computer.setup_confirmed": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","provider":"fixture-provider","confirmed_at":"2026-01-01T00:00:00Z","skills":["fixture-skills"]},
  "computer.setup_finished": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","run_id":"fixture-run_id","confirmed":false,"providers":[{"provider":"fixture-provider","state":"running","status":"fixture-status"}]},
  "computer.setup_turn_activity": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","run_id":"fixture-run_id","turn_id":"fixture-turn_id","provider":"fixture-provider","status":"fixture-status","call_id":"fixture-call_id","kind":"tool_call","tool":"fixture-tool","text":"fixture-text","at":"fixture-at"},
  "computer.setup_turn_changed": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","run_id":"fixture-run_id","turn_id":"fixture-turn_id","provider":"fixture-provider","provider_name":"fixture-provider_name","model":"fixture-model","state":"running","status":"fixture-status","started_at":"2026-01-01T00:00:00Z","ended_at":"2026-01-01T00:00:00Z"},
  "computer.setup_unconfirmed": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","provider":"fixture-provider","confirmed_at":"2026-01-01T00:00:00Z","skills":["fixture-skills"]},
  "computer.tunnel_created": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","tunnel_id":"fixture-tunnel_id","hostname":"fixture-hostname"},
  "computer.tunnel_removed": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","tunnel_id":"fixture-tunnel_id","hostname":"fixture-hostname"},
  "computer.tunnel_status_changed": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","tunnel":"fixture-tunnel","harness_reachable":false,"harness_version":"fixture-harness_version"},
  "deploy.build_completed": {"id":"fixture-id","status":"fixture-status","artifacts":["fixture-artifacts"],"error":"fixture-error"},
  "deploy.build_progress": {"id":"fixture-id","step":1,"total":1,"log":"fixture-log"},
  "deploy.build_started": {"id":"fixture-id","total":1,"log":"fixture-log"},
  "deploy.cancel_requested": {"id":"fixture-id"},
  "deploy.deploy_progress": {"id":"fixture-id","phase":"fixture-phase","log":"fixture-log"},
  "deploy.log": {"id":"fixture-id","phase":"checkout","log":"fixture-log","ts":1},
  "deploy.requested": {"id":"fixture-id","kind":"build","service":"fixture-service","target":"fixture-target","image":"fixture-image","env":{},"strategy":"fixture-strategy","repo":"fixture-repo","ref":"fixture-ref","compose_dir":"fixture-compose_dir","network":"fixture-network","ports":["fixture-ports"],"mounts":["fixture-mounts"],"dockerfile":"fixture-dockerfile","compose_path":"fixture-compose_path","health_check":{}},
  "deploy.status_changed": {"id":"fixture-id","status":"pending","error":"fixture-error"},
  "deploy.updated": {"id":"fixture-id","status":"pending"},
  "dns.exposure_changed": {"exposure_id":"fixture-exposure_id","gateway_id":"fixture-gateway_id","hostname":"fixture-hostname","service":"fixture-service","port":1,"action":"created"},
  "dns.gateway_changed": {"gateway_id":"fixture-gateway_id","kind":"fixture-kind","docker_network":"fixture-docker_network","action":"created"},
  "dns.record_changed": {"zone_id":"fixture-zone_id","zone":"fixture-zone","record_id":"fixture-record_id","action":"created","type":"fixture-type","name":"fixture-name","service":"fixture-service"},
  "dns.tunnel_changed": {"tunnel_id":"fixture-tunnel_id","name":"fixture-name","action":"created","hostname":"fixture-hostname","service":"fixture-service"},
  "doc.created": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","folder_id":"fixture-folder_id","title":"fixture-title","body":"fixture-body","version":1,"archived":false,"locked":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"actor_id":"fixture-actor_id","mentioned_user_ids":["fixture-mentioned_user_ids"]},
  "doc.deleted": {"id":"fixture-id","title":"fixture-title","project_id":"fixture-project_id"},
  "doc.folder.created": {"folder":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","is_default":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"actor_id":"fixture-actor_id"},
  "doc.folder.deleted": {"folder":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","is_default":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"moved_to_folder_id":"fixture-moved_to_folder_id","actor_id":"fixture-actor_id"},
  "doc.folder.updated": {"folder":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","is_default":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"previous_name":"fixture-previous_name","actor_id":"fixture-actor_id"},
  "doc.moved": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","folder_id":"fixture-folder_id","title":"fixture-title"},"from_folder_id":"fixture-from_folder_id","actor_id":"fixture-actor_id"},
  "doc.updated": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","folder_id":"fixture-folder_id","title":"fixture-title","body":"fixture-body","version":1,"archived":false,"locked":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"actor_id":"fixture-actor_id","mentioned_user_ids":["fixture-mentioned_user_ids"]},
  "doc.watchers.changed": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title"},"user_id":"fixture-user_id","watching":false},
  "git.branch_deleted": {"owner":"fixture-owner","repo":"fixture-repo","branch":"fixture-branch"},
  "git.pr_closed": {"owner":"fixture-owner","repo":"fixture-repo","pr":{}},
  "git.pr_comment": {"owner":"fixture-owner","repo":"fixture-repo","pr":{"number":1},"comment":{"body":"fixture-body","author":"fixture-author"}},
  "git.pr_merged": {"owner":"fixture-owner","repo":"fixture-repo","pr":{}},
  "git.pr_opened": {"owner":"fixture-owner","repo":"fixture-repo","pr":{}},
  "git.pr_review_submitted": {"owner":"fixture-owner","repo":"fixture-repo","pr":{}},
  "git.provider_event": {"provider":"fixture-provider","event_type":"fixture-event_type","delivery_id":"fixture-delivery_id","action":"fixture-action","repository":{},"payload":{},"received_at":"2026-01-01T00:00:00Z"},
  "git.push": {"owner":"fixture-owner","repo":"fixture-repo","branch":"fixture-branch","sha":"fixture-sha","pusher":"fixture-pusher"},
  "identity.linked": {"user_id":"fixture-user_id","provider":"fixture-provider","login":"fixture-login"},
  "identity.unlinked": {"user_id":"fixture-user_id","provider":"fixture-provider","login":"fixture-login"},
  "instance.upgrade_changed": {"id":"fixture-id","from_version":"fixture-from_version","to_version":"fixture-to_version","status":"pending","error":"fixture-error","requested_by":"fixture-requested_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},
  "instance.upgrade_requested": {"id":"fixture-id","version":"fixture-version"},
  "instance_template.updated": {"kind":"interview","key":"fixture-key","author_id":"fixture-author_id","updated_at":"2026-01-01T00:00:00Z","reset":false},
  "interview_template.updated": {"workspace_id":"fixture-workspace_id","author_id":"fixture-author_id","updated_at":"2026-01-01T00:00:00Z"},
  "invitation.created": {"invitation_id":"fixture-invitation_id","actor_id":"fixture-actor_id"},
  "invitation.deleted": {"invitation_id":"fixture-invitation_id","reason":"fixture-reason"},
  "invitation.redeemed": {"invitation_id":"fixture-invitation_id","user_id":"fixture-user_id"},
  "invitation.revoked": {"invitation_id":"fixture-invitation_id","actor_id":"fixture-actor_id"},
  "memory.created": {"memory":{"id":"fixture-id","workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","kind":"fixture-kind","title":"fixture-title","when_to_use":"fixture-when_to_use","always_included":false,"updated_at":"2026-01-01T00:00:00Z"},"author_id":"fixture-author_id"},
  "memory.deleted": {"id":"fixture-id","workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","title":"fixture-title","author_id":"fixture-author_id"},
  "memory.updated": {"memory":{"id":"fixture-id","workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","kind":"fixture-kind","title":"fixture-title","when_to_use":"fixture-when_to_use","always_included":false,"updated_at":"2026-01-01T00:00:00Z"},"author_id":"fixture-author_id"},
  "notification.created": {},
  "notification.push_requested": {"notifications":[{"id":"fixture-id","user_id":"fixture-user_id","workspace_id":"fixture-workspace_id"}]},
  "personal_access_token.minted": {"token_id":"fixture-token_id","user_id":"fixture-user_id","name":"fixture-name","computer_id":"fixture-computer_id"},
  "personal_access_token.revoked": {"token_id":"fixture-token_id","user_id":"fixture-user_id","name":"fixture-name","computer_id":"fixture-computer_id"},
  "play.created": {"play":{"id":"fixture-id","workspace_id":"fixture-workspace_id","label":"fixture-label","type":"fixture-type","description":"fixture-description","instructions":"fixture-instructions","enabled":false,"show_when_stage":"fixture-show_when_stage","excluded_project_ids":["fixture-excluded_project_ids"],"created_by":"fixture-created_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "play.deleted": {"id":"fixture-id","label":"fixture-label","workspace_id":"fixture-workspace_id"},
  "play.run_finished": {"trail_id":"fixture-trail_id","play_id":"fixture-play_id","play_label":"fixture-play_label","target_type":"ticket","target_id":"fixture-target_id","target_title":"fixture-target_title","starter_id":"fixture-starter_id","via":"web","workspace_id":"fixture-workspace_id","outcome":"done","last_error":"fixture-last_error","reply_message_id":"fixture-reply_message_id"},
  "play.run_started": {"trail_id":"fixture-trail_id","play_id":"fixture-play_id","play_label":"fixture-play_label","target_type":"ticket","target_id":"fixture-target_id","target_title":"fixture-target_title","starter_id":"fixture-starter_id","via":"web","workspace_id":"fixture-workspace_id","harness_session_id":"fixture-harness_session_id"},
  "play.run_waiting": {"trail_id":"fixture-trail_id","play_id":"fixture-play_id","play_label":"fixture-play_label","target_type":"ticket","target_id":"fixture-target_id","target_title":"fixture-target_title","starter_id":"fixture-starter_id","via":"web","workspace_id":"fixture-workspace_id"},
  "play.updated": {"play":{"id":"fixture-id","workspace_id":"fixture-workspace_id","label":"fixture-label","type":"fixture-type","description":"fixture-description","instructions":"fixture-instructions","enabled":false,"show_when_stage":"fixture-show_when_stage","excluded_project_ids":["fixture-excluded_project_ids"],"created_by":"fixture-created_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "review.status_changed": {"id":"fixture-id","repo":"fixture-repo","pr_number":1,"status":"fixture-status","reviewer":"fixture-reviewer"},
  "role.updated": {"role_id":"fixture-role_id","workspace_id":"fixture-workspace_id","actor_id":"fixture-actor_id"},
  "runner.connected": {"runner_id":"fixture-runner_id","name":"fixture-name"},
  "runner.disconnected": {"runner_id":"fixture-runner_id","reason":"fixture-reason"},
  "runner.heartbeat": {"runner_id":"fixture-runner_id","ts":1},
  "service.created": {"service":{}},
  "service.deleted": {"id":"fixture-id","name":"fixture-name","project_id":"fixture-project_id"},
  "service.updated": {"service":{}},
  "session.created": {"session_id":"fixture-session_id","user_id":"fixture-user_id","client":"browser","platform":"fixture-platform","label":"fixture-label"},
  "session.revoked": {"session_id":"fixture-session_id","user_id":"fixture-user_id","client":"browser","platform":"fixture-platform","label":"fixture-label"},
  "status.created": {"status":{}},
  "status.deleted": {"status":{}},
  "status.updated": {"status":{}},
  "ticket.assignee_changed": {"ticket":{},"from":"fixture-from","to":"fixture-to"},
  "ticket.category_changed": {"ticket_id":"fixture-ticket_id","category_id":"fixture-category_id"},
  "ticket.created": {"ticket":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title","body":"fixture-body","status":"open","doc_id":"fixture-doc_id","assignee":"fixture-assignee","developer":"fixture-developer","tester":"fixture-tester","reporter":{"kind":"user","login":"fixture-login","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:00Z"},"mentioned_user_ids":["fixture-mentioned_user_ids"]},
  "ticket.deleted": {"id":"fixture-id","title":"fixture-title","project_id":"fixture-project_id"},
  "ticket.developer_changed": {"ticket":{},"from":"fixture-from","to":"fixture-to"},
  "ticket.finished": {"ticket":{}},
  "ticket.link_created": {"link":{"ticket_id":"fixture-ticket_id","kind":"found_in","target_id":"fixture-target_id","created_at":"2026-01-01T00:00:00Z"}},
  "ticket.link_deleted": {"link":{"ticket_id":"fixture-ticket_id","kind":"found_in","target_id":"fixture-target_id","created_at":"2026-01-01T00:00:00Z"}},
  "ticket.status_changed": {"ticket":{},"from":"fixture-from","to":"fixture-to","actor":{"kind":"user","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name","play_label":"fixture-play_label","trail_id":"fixture-trail_id","user_id":"fixture-user_id"},"execution_id":"fixture-execution_id"},
  "ticket.test_failed": {"ticket":{},"tester":"fixture-tester","report":"fixture-report"},
  "ticket.test_passed": {"ticket":{},"tester":"fixture-tester"},
  "ticket.tester_changed": {"ticket":{},"from":"fixture-from","to":"fixture-to"},
  "ticket.updated": {"ticket":{},"actor_id":"fixture-actor_id","mentioned_user_ids":["fixture-mentioned_user_ids"]},
  "ticket_type.created": {"ticket_type":{}},
  "ticket_type.deleted": {"ticket_type":{}},
  "ticket_type.updated": {"ticket_type":{}},
  "topology.updated": {"environment":"fixture-environment","canvas":{}},
  "voice.occupancy.changed": {"conversation_id":"fixture-conversation_id","occupants":[{"identity":"fixture-identity","name":"fixture-name"}],"members_only":false},
  "workspace.member.added": {"invitation_id":"fixture-invitation_id","user_id":"fixture-user_id","workspace_id":"fixture-workspace_id","actor_id":"fixture-actor_id"},
  "workspace.member.removed": {"user_id":"fixture-user_id","workspace_id":"fixture-workspace_id","actor_id":"fixture-actor_id"},
  "workspace.member.updated": {"user_id":"fixture-user_id","workspace_id":"fixture-workspace_id","actor_id":"fixture-actor_id"},
  "workspace.updated": {"workspace_id":"fixture-workspace_id","name":"fixture-name","slug":"fixture-slug","actor_id":"fixture-actor_id"},
};
