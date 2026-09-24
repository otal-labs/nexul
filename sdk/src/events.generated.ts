// GENERATED FILE — do not edit by hand.
// Source of truth: internal/integrations/catalog.go (AM11).
// Regenerate: bun run generate:events (from sdk/).

export interface EventPayloads {
  "account.admitted": { "invitation_id"?: string; "user_id"?: string; };
  "account.disabled": { "account_id": string; "actor_id"?: string; };
  "account.reactivated": { "account_id": string; "actor_id"?: string; };
  "account.removed": { "account_id": string; "actor_id"?: string; };
  "account.restored": { "account_id": string; "actor_id"?: string; };
  "category.created": { "category": Record<string, unknown>; };
  "category.deleted": { "category": Record<string, unknown>; };
  "category.updated": { "category": Record<string, unknown>; };
  "chat.conversation.created": { "conversation": Record<string, unknown>; };
  "chat.message.created": { "message": Record<string, unknown>; };
  "chat.message.deleted": { "conversation_id": string; "message_id": string; "deleted_at": string; };
  "chat.message.updated": { "message": Record<string, unknown>; };
  "computer.paired": { "computer_id": string; "user_id": string; "server_url": string; "harness_version"?: string; "token_expires_at": string; };
  "computer.setup_confirmed": { "computer_id": string; "user_id": string; "provider"?: string; "confirmed_at"?: string; "skills"?: string[]; };
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
  "doc.created": { "doc": { "id": string; "title": string; "body"?: string; "version": number; "archived"?: boolean; "created_at"?: string; "updated_at"?: string; }; };
  "doc.deleted": { "id": string; "title": string; };
  "doc.updated": { "doc": { "id": string; "title": string; "body"?: string; "version": number; "archived"?: boolean; "created_at"?: string; "updated_at"?: string; }; };
  "git.branch_deleted": { "owner": string; "repo": string; "branch": string; };
  "git.pr_closed": { "owner": string; "repo": string; "pr": Record<string, unknown>; };
  "git.pr_comment": { "owner": string; "repo": string; "pr": { "number": number; }; "comment": { "body": string; "author": string; }; };
  "git.pr_merged": { "owner": string; "repo": string; "pr": Record<string, unknown>; };
  "git.pr_opened": { "owner": string; "repo": string; "pr": Record<string, unknown>; };
  "git.pr_review_submitted": { "owner": string; "repo": string; "pr": Record<string, unknown>; };
  "git.provider_event": { "provider": string; "event_type": string; "delivery_id": string; "action"?: string; "repository"?: Record<string, unknown>; "payload"?: Record<string, unknown>; "received_at": string; };
  "git.push": { "owner": string; "repo": string; "branch": string; "sha": string; "pusher"?: string; };
  "instance.upgrade_changed": { "id": string; "from_version": string; "to_version": string; "status": "pending" | "started" | "completed" | "failed"; "error"?: string; "requested_by"?: string; "created_at"?: string; "updated_at"?: string; };
  "instance.upgrade_requested": { "id": string; "version": string; };
  "interview_template.updated": { "workspace_id": string; "author_id": string; "updated_at"?: string; };
  "invitation.created": { "invitation_id"?: string; "actor_id"?: string; };
  "invitation.deleted": { "invitation_id"?: string; "reason"?: string; };
  "invitation.redeemed": { "invitation_id"?: string; "user_id"?: string; };
  "invitation.revoked": { "invitation_id"?: string; "actor_id"?: string; };
  "memory.created": { "memory": { "id": string; "workspace_id": string; "project_id": string; "kind"?: string; "title": string; "when_to_use"?: string; "always_included"?: boolean; "updated_at"?: string; }; "author_id": string; };
  "memory.deleted": { "id": string; "title": string; "author_id": string; };
  "memory.updated": { "memory": { "id": string; "workspace_id": string; "project_id": string; "kind"?: string; "title": string; "when_to_use"?: string; "always_included"?: boolean; "updated_at"?: string; }; "author_id": string; };
  "notification.created": Record<string, unknown>;
  "personal_access_token.minted": { "token_id": string; "user_id": string; "name": string; "computer_id"?: string; };
  "personal_access_token.revoked": { "token_id": string; "user_id": string; "name": string; "computer_id"?: string; };
  "play.created": { "play": { "id": string; "workspace_id": string; "label": string; "type": string; "description"?: string; "instructions"?: string; "enabled"?: boolean; "show_when_stage"?: string | null; "excluded_project_ids"?: string[]; "created_by"?: string; "created_at"?: string; "updated_at"?: string; }; };
  "play.deleted": { "id": string; "label": string; };
  "play.run_finished": { "trail_id": string; "play_id": string; "play_label": string; "target_type": "ticket" | "doc"; "target_id": string; "target_title"?: string; "starter_id": string; "via": "web" | "mcp"; "outcome": "done" | "failed" | "interrupted"; "last_error"?: string; "reply_message_id"?: string; };
  "play.run_started": { "trail_id": string; "play_id": string; "play_label": string; "target_type": "ticket" | "doc"; "target_id": string; "target_title"?: string; "starter_id": string; "via": "web" | "mcp"; "harness_session_id"?: string; };
  "play.run_waiting": { "trail_id": string; "play_id": string; "play_label": string; "target_type": "ticket" | "doc"; "target_id": string; "target_title"?: string; "starter_id": string; "via": "web" | "mcp"; };
  "play.updated": { "play": { "id": string; "workspace_id": string; "label": string; "type": string; "description"?: string; "instructions"?: string; "enabled"?: boolean; "show_when_stage"?: string | null; "excluded_project_ids"?: string[]; "created_by"?: string; "created_at"?: string; "updated_at"?: string; }; };
  "review.status_changed": { "id": string; "repo": string; "pr_number": number; "status": string; "reviewer"?: string; };
  "runner.connected": { "runner_id": string; "name"?: string; };
  "runner.disconnected": { "runner_id": string; "reason"?: string; };
  "runner.heartbeat": { "runner_id": string; "ts": number; };
  "service.created": { "service": Record<string, unknown>; };
  "service.deleted": { "id": string; "name": string; };
  "service.updated": { "service": Record<string, unknown>; };
  "status.created": { "status": Record<string, unknown>; };
  "status.deleted": { "status": Record<string, unknown>; };
  "status.updated": { "status": Record<string, unknown>; };
  "ticket.assignee_changed": { "ticket": Record<string, unknown>; "from": string; "to": string; };
  "ticket.category_changed": { "ticket_id": string; "category_id": string; };
  "ticket.created": { "ticket": { "id": string; "project_id"?: string; "title": string; "body"?: string; "status": "open" | "in_progress" | "done" | "closed"; "doc_id"?: string; "assignee"?: string; "developer"?: string; "tester"?: string; "reporter"?: { "kind": "user" | "user:mcp" | "automation"; "login"?: string; "automation_id"?: string; "automation_name"?: string; }; "created_at"?: string; "updated_at"?: string; "finished_at"?: string | null; }; };
  "ticket.deleted": { "id": string; "title": string; };
  "ticket.developer_changed": { "ticket": Record<string, unknown>; "from": string; "to": string; };
  "ticket.finished": { "ticket": Record<string, unknown>; };
  "ticket.link_created": { "link": { "ticket_id": string; "kind": "found_in" | "blocked_by"; "target_id": string; "created_at"?: string; }; };
  "ticket.link_deleted": { "link": { "ticket_id": string; "kind": "found_in" | "blocked_by"; "target_id": string; "created_at"?: string; }; };
  "ticket.status_changed": { "ticket": Record<string, unknown>; "from": string; "to": string; "actor"?: { "kind"?: "user" | "automation" | "play" | "play:mcp"; "automation_id"?: string; "automation_name"?: string; "play_label"?: string; "trail_id"?: string; }; "execution_id"?: string; };
  "ticket.tester_changed": { "ticket": Record<string, unknown>; "from": string; "to": string; };
  "ticket.updated": { "ticket": Record<string, unknown>; };
  "ticket_type.created": { "ticket_type": Record<string, unknown>; };
  "ticket_type.deleted": { "ticket_type": Record<string, unknown>; };
  "ticket_type.updated": { "ticket_type": Record<string, unknown>; };
  "topology.updated": { "environment": string; "canvas"?: Record<string, unknown>; };
  "voice.occupancy.changed": { "conversation_id": string; "occupants": { "identity": string; "name": string; }[]; };
  "workspace.member.added": { "invitation_id"?: string; "user_id"?: string; "workspace_id"?: string; };
}

export type Topic = keyof EventPayloads;

export const TOPICS: Topic[] = [
  "account.admitted",
  "account.disabled",
  "account.reactivated",
  "account.removed",
  "account.restored",
  "category.created",
  "category.deleted",
  "category.updated",
  "chat.conversation.created",
  "chat.message.created",
  "chat.message.deleted",
  "chat.message.updated",
  "computer.paired",
  "computer.setup_confirmed",
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
  "doc.updated",
  "git.branch_deleted",
  "git.pr_closed",
  "git.pr_comment",
  "git.pr_merged",
  "git.pr_opened",
  "git.pr_review_submitted",
  "git.provider_event",
  "git.push",
  "instance.upgrade_changed",
  "instance.upgrade_requested",
  "interview_template.updated",
  "invitation.created",
  "invitation.deleted",
  "invitation.redeemed",
  "invitation.revoked",
  "memory.created",
  "memory.deleted",
  "memory.updated",
  "notification.created",
  "personal_access_token.minted",
  "personal_access_token.revoked",
  "play.created",
  "play.deleted",
  "play.run_finished",
  "play.run_started",
  "play.run_waiting",
  "play.updated",
  "review.status_changed",
  "runner.connected",
  "runner.disconnected",
  "runner.heartbeat",
  "service.created",
  "service.deleted",
  "service.updated",
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
  "ticket.tester_changed",
  "ticket.updated",
  "ticket_type.created",
  "ticket_type.deleted",
  "ticket_type.updated",
  "topology.updated",
  "voice.occupancy.changed",
  "workspace.member.added",
];

export const eventFixtures: { [K in Topic]: EventPayloads[K] } = {
  "account.admitted": {"invitation_id":"fixture-invitation_id","user_id":"fixture-user_id"},
  "account.disabled": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "account.reactivated": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "account.removed": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "account.restored": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "category.created": {"category":{}},
  "category.deleted": {"category":{}},
  "category.updated": {"category":{}},
  "chat.conversation.created": {"conversation":{}},
  "chat.message.created": {"message":{}},
  "chat.message.deleted": {"conversation_id":"fixture-conversation_id","message_id":"fixture-message_id","deleted_at":"2026-01-01T00:00:00Z"},
  "chat.message.updated": {"message":{}},
  "computer.paired": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","server_url":"fixture-server_url","harness_version":"fixture-harness_version","token_expires_at":"2026-01-01T00:00:00Z"},
  "computer.setup_confirmed": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","provider":"fixture-provider","confirmed_at":"2026-01-01T00:00:00Z","skills":["fixture-skills"]},
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
  "doc.created": {"doc":{"id":"fixture-id","title":"fixture-title","body":"fixture-body","version":1,"archived":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "doc.deleted": {"id":"fixture-id","title":"fixture-title"},
  "doc.updated": {"doc":{"id":"fixture-id","title":"fixture-title","body":"fixture-body","version":1,"archived":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "git.branch_deleted": {"owner":"fixture-owner","repo":"fixture-repo","branch":"fixture-branch"},
  "git.pr_closed": {"owner":"fixture-owner","repo":"fixture-repo","pr":{}},
  "git.pr_comment": {"owner":"fixture-owner","repo":"fixture-repo","pr":{"number":1},"comment":{"body":"fixture-body","author":"fixture-author"}},
  "git.pr_merged": {"owner":"fixture-owner","repo":"fixture-repo","pr":{}},
  "git.pr_opened": {"owner":"fixture-owner","repo":"fixture-repo","pr":{}},
  "git.pr_review_submitted": {"owner":"fixture-owner","repo":"fixture-repo","pr":{}},
  "git.provider_event": {"provider":"fixture-provider","event_type":"fixture-event_type","delivery_id":"fixture-delivery_id","action":"fixture-action","repository":{},"payload":{},"received_at":"2026-01-01T00:00:00Z"},
  "git.push": {"owner":"fixture-owner","repo":"fixture-repo","branch":"fixture-branch","sha":"fixture-sha","pusher":"fixture-pusher"},
  "instance.upgrade_changed": {"id":"fixture-id","from_version":"fixture-from_version","to_version":"fixture-to_version","status":"pending","error":"fixture-error","requested_by":"fixture-requested_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},
  "instance.upgrade_requested": {"id":"fixture-id","version":"fixture-version"},
  "interview_template.updated": {"workspace_id":"fixture-workspace_id","author_id":"fixture-author_id","updated_at":"2026-01-01T00:00:00Z"},
  "invitation.created": {"invitation_id":"fixture-invitation_id","actor_id":"fixture-actor_id"},
  "invitation.deleted": {"invitation_id":"fixture-invitation_id","reason":"fixture-reason"},
  "invitation.redeemed": {"invitation_id":"fixture-invitation_id","user_id":"fixture-user_id"},
  "invitation.revoked": {"invitation_id":"fixture-invitation_id","actor_id":"fixture-actor_id"},
  "memory.created": {"memory":{"id":"fixture-id","workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","kind":"fixture-kind","title":"fixture-title","when_to_use":"fixture-when_to_use","always_included":false,"updated_at":"2026-01-01T00:00:00Z"},"author_id":"fixture-author_id"},
  "memory.deleted": {"id":"fixture-id","title":"fixture-title","author_id":"fixture-author_id"},
  "memory.updated": {"memory":{"id":"fixture-id","workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","kind":"fixture-kind","title":"fixture-title","when_to_use":"fixture-when_to_use","always_included":false,"updated_at":"2026-01-01T00:00:00Z"},"author_id":"fixture-author_id"},
  "notification.created": {},
  "personal_access_token.minted": {"token_id":"fixture-token_id","user_id":"fixture-user_id","name":"fixture-name","computer_id":"fixture-computer_id"},
  "personal_access_token.revoked": {"token_id":"fixture-token_id","user_id":"fixture-user_id","name":"fixture-name","computer_id":"fixture-computer_id"},
  "play.created": {"play":{"id":"fixture-id","workspace_id":"fixture-workspace_id","label":"fixture-label","type":"fixture-type","description":"fixture-description","instructions":"fixture-instructions","enabled":false,"show_when_stage":"fixture-show_when_stage","excluded_project_ids":["fixture-excluded_project_ids"],"created_by":"fixture-created_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "play.deleted": {"id":"fixture-id","label":"fixture-label"},
  "play.run_finished": {"trail_id":"fixture-trail_id","play_id":"fixture-play_id","play_label":"fixture-play_label","target_type":"ticket","target_id":"fixture-target_id","target_title":"fixture-target_title","starter_id":"fixture-starter_id","via":"web","outcome":"done","last_error":"fixture-last_error","reply_message_id":"fixture-reply_message_id"},
  "play.run_started": {"trail_id":"fixture-trail_id","play_id":"fixture-play_id","play_label":"fixture-play_label","target_type":"ticket","target_id":"fixture-target_id","target_title":"fixture-target_title","starter_id":"fixture-starter_id","via":"web","harness_session_id":"fixture-harness_session_id"},
  "play.run_waiting": {"trail_id":"fixture-trail_id","play_id":"fixture-play_id","play_label":"fixture-play_label","target_type":"ticket","target_id":"fixture-target_id","target_title":"fixture-target_title","starter_id":"fixture-starter_id","via":"web"},
  "play.updated": {"play":{"id":"fixture-id","workspace_id":"fixture-workspace_id","label":"fixture-label","type":"fixture-type","description":"fixture-description","instructions":"fixture-instructions","enabled":false,"show_when_stage":"fixture-show_when_stage","excluded_project_ids":["fixture-excluded_project_ids"],"created_by":"fixture-created_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "review.status_changed": {"id":"fixture-id","repo":"fixture-repo","pr_number":1,"status":"fixture-status","reviewer":"fixture-reviewer"},
  "runner.connected": {"runner_id":"fixture-runner_id","name":"fixture-name"},
  "runner.disconnected": {"runner_id":"fixture-runner_id","reason":"fixture-reason"},
  "runner.heartbeat": {"runner_id":"fixture-runner_id","ts":1},
  "service.created": {"service":{}},
  "service.deleted": {"id":"fixture-id","name":"fixture-name"},
  "service.updated": {"service":{}},
  "status.created": {"status":{}},
  "status.deleted": {"status":{}},
  "status.updated": {"status":{}},
  "ticket.assignee_changed": {"ticket":{},"from":"fixture-from","to":"fixture-to"},
  "ticket.category_changed": {"ticket_id":"fixture-ticket_id","category_id":"fixture-category_id"},
  "ticket.created": {"ticket":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title","body":"fixture-body","status":"open","doc_id":"fixture-doc_id","assignee":"fixture-assignee","developer":"fixture-developer","tester":"fixture-tester","reporter":{"kind":"user","login":"fixture-login","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:00Z"}},
  "ticket.deleted": {"id":"fixture-id","title":"fixture-title"},
  "ticket.developer_changed": {"ticket":{},"from":"fixture-from","to":"fixture-to"},
  "ticket.finished": {"ticket":{}},
  "ticket.link_created": {"link":{"ticket_id":"fixture-ticket_id","kind":"found_in","target_id":"fixture-target_id","created_at":"2026-01-01T00:00:00Z"}},
  "ticket.link_deleted": {"link":{"ticket_id":"fixture-ticket_id","kind":"found_in","target_id":"fixture-target_id","created_at":"2026-01-01T00:00:00Z"}},
  "ticket.status_changed": {"ticket":{},"from":"fixture-from","to":"fixture-to","actor":{"kind":"user","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name","play_label":"fixture-play_label","trail_id":"fixture-trail_id"},"execution_id":"fixture-execution_id"},
  "ticket.tester_changed": {"ticket":{},"from":"fixture-from","to":"fixture-to"},
  "ticket.updated": {"ticket":{}},
  "ticket_type.created": {"ticket_type":{}},
  "ticket_type.deleted": {"ticket_type":{}},
  "ticket_type.updated": {"ticket_type":{}},
  "topology.updated": {"environment":"fixture-environment","canvas":{}},
  "voice.occupancy.changed": {"conversation_id":"fixture-conversation_id","occupants":[{"identity":"fixture-identity","name":"fixture-name"}]},
  "workspace.member.added": {"invitation_id":"fixture-invitation_id","user_id":"fixture-user_id","workspace_id":"fixture-workspace_id"},
};
