// GENERATED FILE — do not edit by hand.
// Source of truth: internal/eventcatalog/schemas.json (make event-schemas).
// Regenerate: bun run generate:events (from sdk/).

export interface EventPayloads {
  "access.grant.changed": { "resource_type": "doc" | "play" | "project"; "resource_id": string; "user_id": string; "actor_id"?: string; };
  "account.admitted": { "account_id"?: string; "actor_id"?: string; "invitation_id"?: string; "user_id"?: string; "workspace_id"?: string; "reason"?: string; };
  "account.disabled": { "account_id": string; "actor_id"?: string; };
  "account.profile_updated": { "account_id": string; "actor_id"?: string; };
  "account.reactivated": { "account_id": string; "actor_id"?: string; };
  "account.removed": { "account_id": string; "actor_id"?: string; };
  "account.restored": { "account_id": string; "actor_id"?: string; };
  "auto_play.created": { "auto_play": { "id": string; "play_id": string; "workspace_id": string; "enabled": boolean; "moment": "ticket.unblocked" | "ticket.entered_stage" | "ticket.created" | "ticket.developer_set" | "ticket.tester_set" | "ticket.test_failed" | "doc.created" | "doc.changed"; "moment_stage": null | "backlog" | "progress" | "review" | "testing" | "done"; "conditions": { "match": "all" | "any"; "groups": ({ "match": "all" | "any"; "rules": ({ "field": "type" | "project" | "stage" | "status" | "category" | "label" | "developer" | "tester" | "source_doc" | "linked_pr" | "blocked" | "folder"; "op": "is" | "is_not" | "set" | "unset"; "values"?: string[]; })[]; })[]; }; "priority": { "rules": ({ "level": "high" | "normal" | "low"; "when": { "match": "all" | "any"; "rules": ({ "field": "type" | "project" | "stage" | "status" | "category" | "label" | "developer" | "tester" | "source_doc" | "linked_pr" | "blocked" | "folder"; "op": "is" | "is_not" | "set" | "unset"; "values"?: string[]; })[]; }; })[]; "otherwise": "high" | "normal" | "low"; }; "once_within_minutes": number; "run_on": "developer" | "tester" | "causer"; "created_by": string; "created_at": string; "updated_at": string; }; };
  "auto_play.deleted": { "id": string; "play_id": string; "workspace_id": string; };
  "auto_play.limits_updated": { "workspace_id": string; "daily_cap_per_ticket": number; };
  "auto_play.updated": { "auto_play": { "id": string; "play_id": string; "workspace_id": string; "enabled": boolean; "moment": "ticket.unblocked" | "ticket.entered_stage" | "ticket.created" | "ticket.developer_set" | "ticket.tester_set" | "ticket.test_failed" | "doc.created" | "doc.changed"; "moment_stage": null | "backlog" | "progress" | "review" | "testing" | "done"; "conditions": { "match": "all" | "any"; "groups": ({ "match": "all" | "any"; "rules": ({ "field": "type" | "project" | "stage" | "status" | "category" | "label" | "developer" | "tester" | "source_doc" | "linked_pr" | "blocked" | "folder"; "op": "is" | "is_not" | "set" | "unset"; "values"?: string[]; })[]; })[]; }; "priority": { "rules": ({ "level": "high" | "normal" | "low"; "when": { "match": "all" | "any"; "rules": ({ "field": "type" | "project" | "stage" | "status" | "category" | "label" | "developer" | "tester" | "source_doc" | "linked_pr" | "blocked" | "folder"; "op": "is" | "is_not" | "set" | "unset"; "values"?: string[]; })[]; }; })[]; "otherwise": "high" | "normal" | "low"; }; "once_within_minutes": number; "run_on": "developer" | "tester" | "causer"; "created_by": string; "created_at": string; "updated_at": string; }; };
  "botwebhook.created": { "botwebhook_id": string; "conversation_id": string; "workspace_id": string; "name": string; "actor_id"?: string; "changes"?: ("renamed" | "avatar" | "regenerated")[]; "members_only"?: boolean; };
  "botwebhook.deleted": { "botwebhook_id": string; "conversation_id": string; "workspace_id": string; "name": string; "actor_id"?: string; "changes"?: ("renamed" | "avatar" | "regenerated")[]; "members_only"?: boolean; };
  "botwebhook.restored": { "botwebhook_id": string; "conversation_id": string; "workspace_id": string; "name": string; "actor_id"?: string; "changes"?: ("renamed" | "avatar" | "regenerated")[]; "members_only"?: boolean; };
  "botwebhook.updated": { "botwebhook_id": string; "conversation_id": string; "workspace_id": string; "name": string; "actor_id"?: string; "changes"?: ("renamed" | "avatar" | "regenerated")[]; "members_only"?: boolean; };
  "category.created": { "category": { "id": string; "project_id": string; "name": string; "position": number; "color": string; "created_at": string; "updated_at": string; }; };
  "category.deleted": { "category": { "id": string; "project_id": string; "name": string; "position": number; "color": string; "created_at": string; "updated_at": string; }; };
  "category.updated": { "category": { "id": string; "project_id": string; "name": string; "position": number; "color": string; "created_at": string; "updated_at": string; }; };
  "chat.conversation.created": { "conversation": { "id": string; "workspace_id": string; "kind": string; "name"?: string; "ticket_id"?: string; "doc_id"?: string; "project_id"?: string; "parent_message_id"?: string; "created_by": string; "created_at": string; "updated_at": string; "general"?: boolean; "private": boolean; "participant_ids"?: string[]; }; "members_only"?: boolean; };
  "chat.conversation.deleted": { "conversation_id": string; "workspace_id": string; "kind": "channel" | "voice_channel"; "name": string; "actor_id"?: string; "private"?: boolean; "member_ids"?: string[]; "members_only"?: boolean; };
  "chat.conversation.members_changed": { "conversation_id": string; "workspace_id": string; "private": boolean; "added_user_ids": string[]; "removed_user_ids": string[]; "actor_id"?: string; "members_only"?: boolean; };
  "chat.conversation.updated": { "conversation_id": string; "workspace_id": string; "kind": "channel" | "voice_channel"; "name": string; "previous_name": string; "actor_id"?: string; "members_only"?: boolean; };
  "chat.message.created": { "message": { "id": string; "conversation_id": string; "author_id": string; "author_kind": string; "body": string; "mentions": { "kind": string; "handle": string; }[]; "attachment_id"?: string; "edited_at"?: string; "deleted_at"?: string; "created_at": string; "updated_at": string; "reactions"?: { "emoji": string; "user_ids": string[]; }[]; "handoffs"?: { "id": string; "driver": string; "model": string; "title": string; "prompt": string; "state": string; "reply": string; "steps": { "kind": string; "call_id"?: string; "tool"?: string; "summary": string; "detail"?: string; "at": string; }[]; }[]; "via"?: string; "author_name"?: string; "author_avatar_url"?: string; "embeds"?: unknown; }; "members_only"?: boolean; "workspace_id": string; };
  "chat.message.deleted": { "conversation_id": string; "message_id": string; "deleted_at": string; "members_only"?: boolean; "workspace_id": string; };
  "chat.message.reactions_changed": { "conversation_id": string; "message_id": string; "user_id": string; "emoji": string; "reacted": boolean; "members_only"?: boolean; };
  "chat.message.updated": { "message": { "id": string; "conversation_id": string; "author_id": string; "author_kind": string; "body": string; "mentions": { "kind": string; "handle": string; }[]; "attachment_id"?: string; "edited_at"?: string; "deleted_at"?: string; "created_at": string; "updated_at": string; "reactions"?: { "emoji": string; "user_ids": string[]; }[]; "handoffs"?: { "id": string; "driver": string; "model": string; "title": string; "prompt": string; "state": string; "reply": string; "steps": { "kind": string; "call_id"?: string; "tool"?: string; "summary": string; "detail"?: string; "at": string; }[]; }[]; "via"?: string; "author_name"?: string; "author_avatar_url"?: string; "embeds"?: unknown; }; "members_only"?: boolean; };
  "computer.harness_switched": { "computer_id": string; "user_id": string; "from_kind": string; "to_kind": string; "harness_version": string; "members_only": boolean; };
  "computer.paired": { "computer_id": string; "user_id": string; "server_url": string; "harness_version"?: string; "token_expires_at": string; "members_only": boolean; };
  "computer.setup_confirmed": { "computer_id": string; "user_id": string; "provider"?: string; "confirmed_at"?: string; "skills"?: string[]; "members_only": boolean; };
  "computer.setup_finished": { "computer_id": string; "user_id": string; "run_id": string; "confirmed": boolean; "providers": ({ "provider": string; "state": "running" | "confirmed" | "failed"; "status": string; })[]; "members_only": boolean; };
  "computer.setup_turn_activity": { "computer_id": string; "user_id": string; "run_id": string; "turn_id": string; "provider": string; "status": string; "call_id"?: string; "kind"?: "tool_call" | "tool_result" | "text" | "question" | "other" | "note" | "user_message"; "tool"?: string; "text"?: string; "at": string; "members_only": boolean; };
  "computer.setup_turn_changed": { "computer_id": string; "user_id": string; "run_id": string; "turn_id": string; "provider": string; "provider_name": string; "model"?: string; "state": "running" | "confirmed" | "failed"; "status": string; "started_at": string; "ended_at"?: string; "members_only": boolean; };
  "computer.setup_unconfirmed": { "computer_id": string; "user_id": string; "provider"?: string; "confirmed_at"?: string; "skills"?: string[]; "members_only": boolean; };
  "computer.tunnel_created": { "computer_id": string; "user_id": string; "tunnel_id": string; "hostname": string; "members_only": boolean; };
  "computer.tunnel_removed": { "computer_id": string; "user_id": string; "tunnel_id": string; "hostname": string; "members_only": boolean; };
  "computer.tunnel_status_changed": { "computer_id": string; "user_id": string; "tunnel": string; "harness_reachable": boolean; "harness_version"?: string; "members_only": boolean; };
  "deploy.build_completed": { "id": string; "status": string; "artifacts"?: string[]; "error"?: string; };
  "deploy.build_progress": { "id": string; "step": number; "total": number; "log"?: string; };
  "deploy.build_started": { "id": string; "total": number; "log"?: string; };
  "deploy.cancel_requested": { "id": string; };
  "deploy.deploy_progress": { "id": string; "phase": string; "log"?: string; };
  "deploy.log": { "id": string; "phase": "checkout" | "build" | "deploy"; "log": string; "ts": number; };
  "deploy.requested": { "id": string; "kind": "build" | "deploy"; "service"?: string; "target"?: string; "image"?: string; "env"?: Record<string, string>; "strategy"?: string; "repo"?: string; "ref"?: string; "compose_path"?: string; "network"?: string; "ports"?: string[]; "mounts"?: string[]; "command"?: string[]; "dockerfile"?: string; "stack_slug"?: string; "stack_root"?: string; "gateway_container"?: string; "join_networks"?: string[]; };
  "deploy.status_changed": { "id": string; "status": "pending" | "running" | "healthy" | "failed"; "error"?: string; "address"?: string; "services"?: { "name": string; "container_name": string; "image"?: string; "status": string; "networks"?: { "name": string; "address"?: string; "gateway_container"?: string; "join_networks"?: string[]; }[]; "ports"?: string[]; }[]; };
  "deploy.updated": { "id": string; "status": "pending" | "running" | "healthy" | "failed"; "stack_id": string; };
  "dns.exposure_changed": { "exposure_id": string; "gateway_id": string; "hostname"?: string; "service"?: string; "port"?: number; "action": "created" | "deleted"; };
  "dns.gateway_changed": { "gateway_id": string; "kind"?: string; "docker_network"?: string; "action": "created" | "deleted"; };
  "dns.record_changed": { "zone_id": string; "zone"?: string; "record_id"?: string; "action": "created" | "updated" | "deleted"; "type"?: string; "name"?: string; "service"?: string; };
  "dns.tunnel_changed": { "tunnel_id": string; "name"?: string; "action": "created" | "routed" | "rotated" | "deleted"; "hostname"?: string; "service"?: string; };
  "doc.clarification.answer_cleared": { "doc": { "id": string; "project_id": string; "title": string; }; "round": number; "question_id": string; "question": string; "author_id": string; "at": string; };
  "doc.clarification.answer_saved": { "doc": { "id": string; "project_id": string; "title": string; }; "round": number; "question_id": string; "question": string; "author_id": string; "at": string; };
  "doc.clarification.anything_else_saved": { "doc": { "id": string; "project_id": string; "title": string; }; "round": number; "started_by": string; "actor_id"?: string; "removed"?: boolean; };
  "doc.clarification.closed": { "doc": { "id": string; "project_id": string; "title": string; }; "round": number; "started_by": string; "actor_id"?: string; "removed"?: boolean; };
  "doc.clarification.round_answered": { "doc": { "id": string; "project_id": string; "title": string; }; "round": number; "started_by": string; "actor_id"?: string; "removed"?: boolean; };
  "doc.clarification.round_ended": { "doc": { "id": string; "project_id": string; "title": string; }; "round": number; "started_by": string; "actor_id"?: string; "removed"?: boolean; };
  "doc.clarification.round_posted": { "doc": { "id": string; "project_id": string; "title": string; }; "round": number; "started_by": string; "actor_id"?: string; "removed"?: boolean; "question_count": number; "no_gaps": boolean; };
  "doc.clarification.round_started": { "doc": { "id": string; "project_id": string; "title": string; }; "round": number; "started_by": string; "actor_id"?: string; "removed"?: boolean; };
  "doc.created": { "doc": { "id": string; "title": string; "body": string; "project_id": string; "folder_id": string; "version": number; "archived": boolean; "locked": boolean; "created_by": string; "created_at": string; "updated_at": string; }; "actor_id"?: string; "mentioned_user_ids"?: string[]; };
  "doc.deleted": { "id": string; "title": string; "project_id": string; };
  "doc.folder.created": { "folder": { "id": string; "project_id": string; "name": string; "is_default": boolean; "created_at": string; "updated_at": string; }; "actor_id"?: string; };
  "doc.folder.deleted": { "folder": { "id": string; "project_id": string; "name": string; "is_default": boolean; "created_at": string; "updated_at": string; }; "moved_to_folder_id": string; "actor_id"?: string; };
  "doc.folder.updated": { "folder": { "id": string; "project_id": string; "name": string; "is_default": boolean; "created_at": string; "updated_at": string; }; "previous_name": string; "actor_id"?: string; };
  "doc.moved": { "doc": { "id": string; "title": string; "body": string; "project_id": string; "folder_id": string; "version": number; "archived": boolean; "locked": boolean; "created_by": string; "created_at": string; "updated_at": string; }; "from_folder_id": string; "actor_id"?: string; };
  "doc.settled": { "doc": { "id": string; "project_id": string; "title": string; }; "first": boolean; "actor_id": string; };
  "doc.updated": { "doc": { "id": string; "title": string; "body": string; "project_id": string; "folder_id": string; "version": number; "archived": boolean; "locked": boolean; "created_by": string; "created_at": string; "updated_at": string; }; "actor_id"?: string; "mentioned_user_ids"?: string[]; "lock_changed"?: boolean; };
  "doc.watchers.changed": { "doc": { "id": string; "project_id": string; "title": string; }; "user_id": string; "watching": boolean; };
  "git.branch_deleted": { "owner": string; "repo": string; "branch": string; };
  "git.pr_closed": { "owner": string; "repo": string; "pr": { "number": number; "title": string; "body": string; "state": string; "merged": boolean; "head_sha": string; "base_branch": string; "author": string; "linked_ticket_ids": string[]; }; };
  "git.pr_comment": { "owner": string; "repo": string; "pr": { "number": number; }; "comment": { "body": string; "author": string; }; };
  "git.pr_merged": { "owner": string; "repo": string; "pr": { "number": number; "title": string; "body": string; "state": string; "merged": boolean; "head_sha": string; "base_branch": string; "author": string; "linked_ticket_ids": string[]; }; };
  "git.pr_opened": { "owner": string; "repo": string; "pr": { "number": number; "title": string; "body": string; "state": string; "merged": boolean; "head_sha": string; "base_branch": string; "author": string; "linked_ticket_ids": string[]; }; };
  "git.pr_review_submitted": { "owner": string; "repo": string; "pr": { "number": number; "state": string; "reviewer": string; }; };
  "git.provider_event": { "provider": string; "event_type": string; "delivery_id": string; "action"?: string; "repository"?: { "id": number; "name": string; "full_name": string; "owner": string; "html_url": string; "default_branch": string; }; "payload": Record<string, unknown>; "received_at": string; };
  "git.push": { "owner": string; "repo": string; "branch": string; "sha": string; "pusher"?: string; };
  "identity.linked": { "user_id": string; "provider": string; "login": string; };
  "identity.unlinked": { "user_id": string; "provider": string; "login": string; };
  "instance.upgrade_changed": { "id": string; "from_version": string; "to_version": string; "status": "pending" | "started" | "completed" | "failed"; "error": string; "requested_by": string; "created_at": string; "updated_at": string; };
  "instance.upgrade_requested": { "id": string; "version": string; };
  "instance_template.updated": { "kind": "interview" | "mention_chip" | "play_instructions" | "ticket_body" | "agent_prompt"; "key": string; "author_id": string; "updated_at": string; "reset": boolean; };
  "interview_answer.cleared": { "workspace_id": string; "project_id": string; "round": number; "question": string; "author_id": string; "at": string; };
  "interview_answer.saved": { "workspace_id": string; "project_id": string; "round": number; "question": string; "author_id": string; "at": string; };
  "interview_draft.dismissed": { "workspace_id": string; "project_id": string; "draft_id": string; "question": string; "author_id": string; "at": string; };
  "interview_draft.saved": { "workspace_id": string; "project_id": string; "draft_id": string; "question": string; "author_id": string; "at": string; };
  "interview_source.added": { "workspace_id": string; "project_id": string; "source_id": string; "kind": "path" | "doc" | "memory" | "project" | "text"; "stance": "follow" | "question"; "author_id": string; "at": string; };
  "interview_source.changed": { "workspace_id": string; "project_id": string; "source_id": string; "kind": "path" | "doc" | "memory" | "project" | "text"; "stance": "follow" | "question"; "author_id": string; "at": string; };
  "interview_source.removed": { "workspace_id": string; "project_id": string; "source_id": string; "kind": "path" | "doc" | "memory" | "project" | "text"; "stance": "follow" | "question"; "author_id": string; "at": string; };
  "interview_template.updated": { "workspace_id": string; "author_id": string; "updated_at": string; };
  "invitation.created": { "invitation_id": string; "actor_id"?: string; "user_id"?: string; "workspace_id"?: string; "reason"?: string; };
  "invitation.deleted": { "invitation_id": string; "actor_id"?: string; "user_id"?: string; "workspace_id"?: string; "reason"?: string; };
  "invitation.redeemed": { "invitation_id": string; "actor_id"?: string; "user_id"?: string; "workspace_id"?: string; "reason"?: string; };
  "invitation.revoked": { "invitation_id": string; "actor_id"?: string; "user_id"?: string; "workspace_id"?: string; "reason"?: string; };
  "memory.created": { "memory": { "id": string; "workspace_id": string; "project_id": string; "kind"?: string; "title": string; "when_to_use": string; "always_included": boolean; "footer": boolean; "version": number; "updated_at": string; }; "author_id": string; };
  "memory.deleted": { "id": string; "workspace_id": string; "project_id": string; "title": string; "author_id": string; };
  "memory.updated": { "memory": { "id": string; "workspace_id": string; "project_id": string; "kind"?: string; "title": string; "when_to_use": string; "always_included": boolean; "footer": boolean; "version": number; "updated_at": string; }; "author_id": string; "author_via"?: string; };
  "notification.created": { "user_ids": string[]; "workspace_id": string; "project_id"?: string; };
  "notification.push_requested": { "notifications": { "id": string; "user_id": string; "workspace_id"?: string; }[]; };
  "personal_access_token.minted": { "token_id": string; "user_id": string; "name": string; "computer_id"?: string; };
  "personal_access_token.revoked": { "token_id": string; "user_id": string; "name": string; "computer_id"?: string; };
  "play.created": { "play": { "id": string; "workspace_id": string; "label": string; "type": string; "description": string; "instructions": string; "enabled": boolean; "show_when_stage": null | string; "excluded_project_ids": string[]; "builtin_key": string; "created_by": string; "created_at": string; "updated_at": string; }; };
  "play.deleted": { "id": string; "label": string; "workspace_id": string; };
  "play.queue_resumed": { "workspace_id": string; "project_id": string; "target_type": "ticket" | "doc"; "target_id": string; "resumed_by": string; "resumed_at": string; };
  "play.queue_updated": { "id": string; "workspace_id": string; "project_id": string; "target_type": "ticket" | "doc"; "target_id": string; "play_id": string; "play_label": string; "auto_play_id": string; "automation_id": string; "person_id": string; "run_on": "developer" | "tester" | "causer"; "moment": "ticket.unblocked" | "ticket.entered_stage" | "ticket.created" | "ticket.developer_set" | "ticket.tester_set" | "ticket.test_failed" | "doc.created" | "doc.changed" | "automation"; "priority": "high" | "normal" | "low"; "status": "queued" | "dispatching" | "started" | "skipped" | "didnt_run" | "cancelled"; "reason": string; "trail_id": string; "via": "web" | "mcp" | "automation"; "queued_at": string; "decided_at": null | string; "not_before": string; };
  "play.queued": { "id": string; "workspace_id": string; "project_id": string; "target_type": "ticket" | "doc"; "target_id": string; "play_id": string; "play_label": string; "auto_play_id": string; "automation_id": string; "person_id": string; "run_on": "developer" | "tester" | "causer"; "moment": "ticket.unblocked" | "ticket.entered_stage" | "ticket.created" | "ticket.developer_set" | "ticket.tester_set" | "ticket.test_failed" | "doc.created" | "doc.changed" | "automation"; "priority": "high" | "normal" | "low"; "status": "queued" | "dispatching" | "started" | "skipped" | "didnt_run" | "cancelled"; "reason": string; "trail_id": string; "via": "web" | "mcp" | "automation"; "queued_at": string; "decided_at": null | string; "not_before": string; };
  "play.run_finished": { "trail_id": string; "play_id": string; "play_label": string; "target_type": "ticket" | "doc" | "interview"; "target_id": string; "target_title": string; "starter_id": string; "via": "web" | "mcp"; "workspace_id"?: string; "outcome": "done" | "failed" | "interrupted"; "last_error"?: string; "reply_message_id"?: string; };
  "play.run_started": { "trail_id": string; "play_id": string; "play_label": string; "target_type": "ticket" | "doc" | "interview"; "target_id": string; "target_title": string; "starter_id": string; "via": "web" | "mcp"; "workspace_id"?: string; "harness_session_id": string; };
  "play.run_waiting": { "trail_id": string; "play_id": string; "play_label": string; "target_type": "ticket" | "doc" | "interview"; "target_id": string; "target_title": string; "starter_id": string; "via": "web" | "mcp"; "workspace_id"?: string; };
  "play.updated": { "play": { "id": string; "workspace_id": string; "label": string; "type": string; "description": string; "instructions": string; "enabled": boolean; "show_when_stage": null | string; "excluded_project_ids": string[]; "builtin_key": string; "created_by": string; "created_at": string; "updated_at": string; }; };
  "project.setup_changed": { "project_id": string; "workspace_id": string; "setup": { "stack_id"?: string; "env_keys"?: string[]; "finished": boolean; "steps": Record<string, string>; }; };
  "repository.installation.assigned": { "account_id": number; "account_login": string; "workspace_id": string; "actor_id"?: string; "uninstalled"?: boolean; };
  "repository.installation.unassigned": { "account_id": number; "account_login": string; "workspace_id": string; "actor_id"?: string; "uninstalled"?: boolean; };
  "review.status_changed": { "id": string; "repo": string; "pr_number": number; "status": string; "reviewer"?: string; };
  "role.updated": { "role_id": string; "workspace_id": string; "actor_id"?: string; };
  "runner.connected": { "runner_id": string; "name"?: string; };
  "runner.disconnected": { "runner_id": string; "reason"?: string; };
  "runner.facts_reported": { "runner_id": string; "computer_id": string; "user_id": string; "facts": { "hostname"?: string; "t3": { "state": "answering" | "not_running" | "missing" | "not_loopback"; "port"?: number; "version"?: string; }; }; "members_only": boolean; };
  "runner.heartbeat": { "runner_id": string; "ts": number; };
  "runner.personal_changed": { "runner_id": string; "computer_id": string; "user_id": string; "state": "enrolled" | "connected" | "disconnected" | "removed"; "hostname"?: string; "members_only": boolean; };
  "service.created": { "stack": { "id": string; "project_id": string; "name": string; "slug": string; "machine": string; "strategy": string; "compose_path"?: string; "env"?: Record<string, string>; "docker_network"?: string; "ports"?: string[]; "mounts"?: string[]; "command"?: string[]; "build_source"?: { "repo_owner"?: string; "repo_name"?: string; "branch"?: string; "dockerfile"?: string; "compose_path"?: string; }; "branch_deploy_rules"?: { "pattern": string; "docker_network": string; "hostname_template"?: string; "name_suffix"?: string; "port"?: number; "overrides"?: Record<string, string>; }[]; "derived_from"?: string; "branch"?: string; "managed": boolean; "created_at": string; "updated_at": string; }; "services"?: { "id": string; "stack_id": string; "name": string; "declared": { "image"?: string; "build"?: string; "ports"?: string[]; "env_keys"?: string[]; }; "container_name"?: string; "image"?: string; "status": string; "networks"?: { "name": string; "address"?: string; }[]; "ports"?: string[]; "observed_at"?: string; }[]; };
  "service.deleted": { "id": string; "name": string; "service_ids"?: string[]; "project_id"?: string; };
  "service.updated": { "stack": { "id": string; "project_id": string; "name": string; "slug": string; "machine": string; "strategy": string; "compose_path"?: string; "env"?: Record<string, string>; "docker_network"?: string; "ports"?: string[]; "mounts"?: string[]; "command"?: string[]; "build_source"?: { "repo_owner"?: string; "repo_name"?: string; "branch"?: string; "dockerfile"?: string; "compose_path"?: string; }; "branch_deploy_rules"?: { "pattern": string; "docker_network": string; "hostname_template"?: string; "name_suffix"?: string; "port"?: number; "overrides"?: Record<string, string>; }[]; "derived_from"?: string; "branch"?: string; "managed": boolean; "created_at": string; "updated_at": string; }; "services"?: { "id": string; "stack_id": string; "name": string; "declared": { "image"?: string; "build"?: string; "ports"?: string[]; "env_keys"?: string[]; }; "container_name"?: string; "image"?: string; "status": string; "networks"?: { "name": string; "address"?: string; }[]; "ports"?: string[]; "observed_at"?: string; }[]; };
  "session.created": { "session_id": string; "user_id": string; "client": "browser" | "desktop" | "phone"; "platform": string; "label": string; };
  "session.revoked": { "session_id": string; "user_id": string; "client": "browser" | "desktop" | "phone"; "platform": string; "label": string; };
  "status.created": { "status": { "id": string; "project_id": string; "name": string; "position": number; "kind": string; "icon": string; "created_at": string; "updated_at": string; }; "previous_kind"?: string; };
  "status.deleted": { "status": { "id": string; "project_id": string; "name": string; "position": number; "kind": string; "icon": string; "created_at": string; "updated_at": string; }; "previous_kind"?: string; };
  "status.updated": { "status": { "id": string; "project_id": string; "name": string; "position": number; "kind": string; "icon": string; "created_at": string; "updated_at": string; }; "previous_kind"?: string; };
  "ticket.assignee_changed": { "ticket": { "id": string; "project_id": string; "category_id": string; "type_id": string; "title": string; "body": string; "status": string; "position": number; "number": number; "doc_id": string; "developer": string; "tester": string; "reporter": { "kind": "user" | "user:mcp" | "automation"; "login"?: string; "automation_id"?: string; "automation_name"?: string; }; "created_at": string; "updated_at": string; "finished_at"?: string; "labels": string[]; "assignee": string; }; "from": string; "to": string; };
  "ticket.category_changed": { "ticket_id": string; "category_id": string; "project_id"?: string; };
  "ticket.created": { "ticket": { "id": string; "project_id": string; "category_id": string; "type_id": string; "title": string; "body": string; "status": string; "position": number; "number": number; "doc_id": string; "developer": string; "tester": string; "reporter": { "kind": "user" | "user:mcp" | "automation"; "login"?: string; "automation_id"?: string; "automation_name"?: string; }; "created_at": string; "updated_at": string; "finished_at"?: string; "labels": string[]; "assignee": string; }; "mentioned_user_ids"?: string[]; };
  "ticket.deleted": { "id": string; "title": string; "project_id": string; };
  "ticket.developer_changed": { "ticket": { "id": string; "project_id": string; "category_id": string; "type_id": string; "title": string; "body": string; "status": string; "position": number; "number": number; "doc_id": string; "developer": string; "tester": string; "reporter": { "kind": "user" | "user:mcp" | "automation"; "login"?: string; "automation_id"?: string; "automation_name"?: string; }; "created_at": string; "updated_at": string; "finished_at"?: string; "labels": string[]; "assignee": string; }; "from": string; "to": string; "actor"?: { "kind": "user" | "user:mcp" | "automation" | "play" | "play:mcp"; "automation_id"?: string; "automation_name"?: string; "play_label"?: string; "trail_id"?: string; "user_id"?: string; }; };
  "ticket.finished": { "ticket": { "id": string; "project_id": string; "category_id": string; "type_id": string; "title": string; "body": string; "status": string; "position": number; "number": number; "doc_id": string; "developer": string; "tester": string; "reporter": { "kind": "user" | "user:mcp" | "automation"; "login"?: string; "automation_id"?: string; "automation_name"?: string; }; "created_at": string; "updated_at": string; "finished_at"?: string; "labels": string[]; "assignee": string; }; };
  "ticket.link_created": { "link": { "ticket_id": string; "kind": "found_in" | "blocked_by"; "target_id": string; "created_at": string; }; };
  "ticket.link_deleted": { "link": { "ticket_id": string; "kind": "found_in" | "blocked_by"; "target_id": string; "created_at": string; }; };
  "ticket.status_changed": { "ticket": { "id": string; "project_id": string; "category_id": string; "type_id": string; "title": string; "body": string; "status": string; "position": number; "number": number; "doc_id": string; "developer": string; "tester": string; "reporter": { "kind": "user" | "user:mcp" | "automation"; "login"?: string; "automation_id"?: string; "automation_name"?: string; }; "created_at": string; "updated_at": string; "finished_at"?: string; "labels": string[]; "assignee": string; }; "from": string; "to": string; "actor"?: { "kind": "user" | "user:mcp" | "automation" | "play" | "play:mcp"; "automation_id"?: string; "automation_name"?: string; "play_label"?: string; "trail_id"?: string; "user_id"?: string; }; "run_id"?: string; };
  "ticket.test_failed": { "ticket": { "id": string; "project_id": string; "category_id": string; "type_id": string; "title": string; "body": string; "status": string; "position": number; "number": number; "doc_id": string; "developer": string; "tester": string; "reporter": { "kind": "user" | "user:mcp" | "automation"; "login"?: string; "automation_id"?: string; "automation_name"?: string; }; "created_at": string; "updated_at": string; "finished_at"?: string; "labels": string[]; "assignee": string; }; "tester": string; "report"?: string; };
  "ticket.test_passed": { "ticket": { "id": string; "project_id": string; "category_id": string; "type_id": string; "title": string; "body": string; "status": string; "position": number; "number": number; "doc_id": string; "developer": string; "tester": string; "reporter": { "kind": "user" | "user:mcp" | "automation"; "login"?: string; "automation_id"?: string; "automation_name"?: string; }; "created_at": string; "updated_at": string; "finished_at"?: string; "labels": string[]; "assignee": string; }; "tester": string; "report"?: string; };
  "ticket.tester_changed": { "ticket": { "id": string; "project_id": string; "category_id": string; "type_id": string; "title": string; "body": string; "status": string; "position": number; "number": number; "doc_id": string; "developer": string; "tester": string; "reporter": { "kind": "user" | "user:mcp" | "automation"; "login"?: string; "automation_id"?: string; "automation_name"?: string; }; "created_at": string; "updated_at": string; "finished_at"?: string; "labels": string[]; "assignee": string; }; "from": string; "to": string; "actor"?: { "kind": "user" | "user:mcp" | "automation" | "play" | "play:mcp"; "automation_id"?: string; "automation_name"?: string; "play_label"?: string; "trail_id"?: string; "user_id"?: string; }; };
  "ticket.unblocked": { "ticket_id": string; "project_id": string; "blocker_id": string; "cause": "blocker_done" | "link_deleted" | "blocker_deleted"; "actor": { "kind": "user" | "user:mcp" | "automation" | "play" | "play:mcp"; "automation_id"?: string; "automation_name"?: string; "play_label"?: string; "trail_id"?: string; "user_id"?: string; }; };
  "ticket.updated": { "ticket": { "id": string; "project_id": string; "category_id": string; "type_id": string; "title": string; "body": string; "status": string; "position": number; "number": number; "doc_id": string; "developer": string; "tester": string; "reporter": { "kind": "user" | "user:mcp" | "automation"; "login"?: string; "automation_id"?: string; "automation_name"?: string; }; "created_at": string; "updated_at": string; "finished_at"?: string; "labels": string[]; "assignee": string; }; "actor_id"?: string; "mentioned_user_ids"?: string[]; };
  "ticket_type.created": { "ticket_type": { "id": string; "project_id": string; "name": string; "position": number; "color": string; "body_template": string; "created_at": string; "updated_at": string; }; };
  "ticket_type.deleted": { "ticket_type": { "id": string; "project_id": string; "name": string; "position": number; "color": string; "body_template": string; "created_at": string; "updated_at": string; }; };
  "ticket_type.updated": { "ticket_type": { "id": string; "project_id": string; "name": string; "position": number; "color": string; "body_template": string; "created_at": string; "updated_at": string; }; };
  "topology.updated": { "environment": string; "workspace_id"?: string; "canvas": { "schema_version": number; "nodes": { "id": string; "type": string; "position": { "x": number; "y": number; }; "data": { "service_id"?: string; "name"?: string; "runtime"?: string; "url"?: string; "status"?: string; "replicas"?: number; "volume"?: string; "address"?: string; "label"?: string; }; }[]; "edges": { "id": string; "source": string; "target": string; "type": string; "data": { "kind": string; }; }[]; "viewport"?: { "x": number; "y": number; "zoom": number; }; }; };
  "voice.occupancy.changed": { "conversation_id": string; "occupants": { "identity": string; "name": string; }[]; "members_only"?: boolean; };
  "workspace.member.added": { "invitation_id"?: string; "actor_id"?: string; "user_id"?: string; "workspace_id"?: string; "reason"?: string; "project_ids"?: string[]; };
  "workspace.member.removed": { "user_id": string; "workspace_id": string; "actor_id"?: string; "project_ids"?: string[]; };
  "workspace.member.updated": { "user_id": string; "workspace_id": string; "actor_id"?: string; "project_ids"?: string[]; };
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
  "auto_play.created",
  "auto_play.deleted",
  "auto_play.limits_updated",
  "auto_play.updated",
  "botwebhook.created",
  "botwebhook.deleted",
  "botwebhook.restored",
  "botwebhook.updated",
  "category.created",
  "category.deleted",
  "category.updated",
  "chat.conversation.created",
  "chat.conversation.deleted",
  "chat.conversation.members_changed",
  "chat.conversation.updated",
  "chat.message.created",
  "chat.message.deleted",
  "chat.message.reactions_changed",
  "chat.message.updated",
  "computer.harness_switched",
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
  "doc.clarification.answer_cleared",
  "doc.clarification.answer_saved",
  "doc.clarification.anything_else_saved",
  "doc.clarification.closed",
  "doc.clarification.round_answered",
  "doc.clarification.round_ended",
  "doc.clarification.round_posted",
  "doc.clarification.round_started",
  "doc.created",
  "doc.deleted",
  "doc.folder.created",
  "doc.folder.deleted",
  "doc.folder.updated",
  "doc.moved",
  "doc.settled",
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
  "interview_answer.cleared",
  "interview_answer.saved",
  "interview_draft.dismissed",
  "interview_draft.saved",
  "interview_source.added",
  "interview_source.changed",
  "interview_source.removed",
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
  "play.queue_resumed",
  "play.queue_updated",
  "play.queued",
  "play.run_finished",
  "play.run_started",
  "play.run_waiting",
  "play.updated",
  "project.setup_changed",
  "repository.installation.assigned",
  "repository.installation.unassigned",
  "review.status_changed",
  "role.updated",
  "runner.connected",
  "runner.disconnected",
  "runner.facts_reported",
  "runner.heartbeat",
  "runner.personal_changed",
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
  "ticket.unblocked",
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
  "account.admitted": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id","invitation_id":"fixture-invitation_id","user_id":"fixture-user_id","workspace_id":"fixture-workspace_id","reason":"fixture-reason"},
  "account.disabled": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "account.profile_updated": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "account.reactivated": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "account.removed": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "account.restored": {"account_id":"fixture-account_id","actor_id":"fixture-actor_id"},
  "auto_play.created": {"auto_play":{"id":"fixture-id","play_id":"fixture-play_id","workspace_id":"fixture-workspace_id","enabled":false,"moment":"ticket.unblocked","moment_stage":"backlog","conditions":{"match":"all","groups":[{"match":"all","rules":[{"field":"type","op":"is","values":["fixture-values"]}]}]},"priority":{"rules":[{"level":"high","when":{"match":"all","rules":[{"field":"type","op":"is","values":["fixture-values"]}]}}],"otherwise":"high"},"once_within_minutes":1,"run_on":"developer","created_by":"fixture-created_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "auto_play.deleted": {"id":"fixture-id","play_id":"fixture-play_id","workspace_id":"fixture-workspace_id"},
  "auto_play.limits_updated": {"workspace_id":"fixture-workspace_id","daily_cap_per_ticket":1},
  "auto_play.updated": {"auto_play":{"id":"fixture-id","play_id":"fixture-play_id","workspace_id":"fixture-workspace_id","enabled":false,"moment":"ticket.unblocked","moment_stage":"backlog","conditions":{"match":"all","groups":[{"match":"all","rules":[{"field":"type","op":"is","values":["fixture-values"]}]}]},"priority":{"rules":[{"level":"high","when":{"match":"all","rules":[{"field":"type","op":"is","values":["fixture-values"]}]}}],"otherwise":"high"},"once_within_minutes":1,"run_on":"developer","created_by":"fixture-created_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "botwebhook.created": {"botwebhook_id":"fixture-botwebhook_id","conversation_id":"fixture-conversation_id","workspace_id":"fixture-workspace_id","name":"fixture-name","actor_id":"fixture-actor_id","changes":["renamed"],"members_only":false},
  "botwebhook.deleted": {"botwebhook_id":"fixture-botwebhook_id","conversation_id":"fixture-conversation_id","workspace_id":"fixture-workspace_id","name":"fixture-name","actor_id":"fixture-actor_id","changes":["renamed"],"members_only":false},
  "botwebhook.restored": {"botwebhook_id":"fixture-botwebhook_id","conversation_id":"fixture-conversation_id","workspace_id":"fixture-workspace_id","name":"fixture-name","actor_id":"fixture-actor_id","changes":["renamed"],"members_only":false},
  "botwebhook.updated": {"botwebhook_id":"fixture-botwebhook_id","conversation_id":"fixture-conversation_id","workspace_id":"fixture-workspace_id","name":"fixture-name","actor_id":"fixture-actor_id","changes":["renamed"],"members_only":false},
  "category.created": {"category":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","position":1,"color":"fixture-color","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "category.deleted": {"category":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","position":1,"color":"fixture-color","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "category.updated": {"category":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","position":1,"color":"fixture-color","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "chat.conversation.created": {"conversation":{"id":"fixture-id","workspace_id":"fixture-workspace_id","kind":"fixture-kind","name":"fixture-name","ticket_id":"fixture-ticket_id","doc_id":"fixture-doc_id","project_id":"fixture-project_id","parent_message_id":"fixture-parent_message_id","created_by":"fixture-created_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","general":false,"private":false,"participant_ids":["fixture-participant_ids"]},"members_only":false},
  "chat.conversation.deleted": {"conversation_id":"fixture-conversation_id","workspace_id":"fixture-workspace_id","kind":"channel","name":"fixture-name","actor_id":"fixture-actor_id","private":false,"member_ids":["fixture-member_ids"],"members_only":false},
  "chat.conversation.members_changed": {"conversation_id":"fixture-conversation_id","workspace_id":"fixture-workspace_id","private":false,"added_user_ids":["fixture-added_user_ids"],"removed_user_ids":["fixture-removed_user_ids"],"actor_id":"fixture-actor_id","members_only":false},
  "chat.conversation.updated": {"conversation_id":"fixture-conversation_id","workspace_id":"fixture-workspace_id","kind":"channel","name":"fixture-name","previous_name":"fixture-previous_name","actor_id":"fixture-actor_id","members_only":false},
  "chat.message.created": {"message":{"id":"fixture-id","conversation_id":"fixture-conversation_id","author_id":"fixture-author_id","author_kind":"fixture-author_kind","body":"fixture-body","mentions":[{"kind":"fixture-kind","handle":"fixture-handle"}],"attachment_id":"fixture-attachment_id","edited_at":"2026-01-01T00:00:00Z","deleted_at":"2026-01-01T00:00:00Z","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","reactions":[{"emoji":"fixture-emoji","user_ids":["fixture-user_ids"]}],"handoffs":[{"id":"fixture-id","driver":"fixture-driver","model":"fixture-model","title":"fixture-title","prompt":"fixture-prompt","state":"fixture-state","reply":"fixture-reply","steps":[{"kind":"fixture-kind","call_id":"fixture-call_id","tool":"fixture-tool","summary":"fixture-summary","detail":"fixture-detail","at":"2026-01-01T00:00:00Z"}]}],"via":"fixture-via","author_name":"fixture-author_name","author_avatar_url":"fixture-author_avatar_url","embeds":null},"members_only":false,"workspace_id":"fixture-workspace_id"},
  "chat.message.deleted": {"conversation_id":"fixture-conversation_id","message_id":"fixture-message_id","deleted_at":"2026-01-01T00:00:00Z","members_only":false,"workspace_id":"fixture-workspace_id"},
  "chat.message.reactions_changed": {"conversation_id":"fixture-conversation_id","message_id":"fixture-message_id","user_id":"fixture-user_id","emoji":"fixture-emoji","reacted":false,"members_only":false},
  "chat.message.updated": {"message":{"id":"fixture-id","conversation_id":"fixture-conversation_id","author_id":"fixture-author_id","author_kind":"fixture-author_kind","body":"fixture-body","mentions":[{"kind":"fixture-kind","handle":"fixture-handle"}],"attachment_id":"fixture-attachment_id","edited_at":"2026-01-01T00:00:00Z","deleted_at":"2026-01-01T00:00:00Z","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","reactions":[{"emoji":"fixture-emoji","user_ids":["fixture-user_ids"]}],"handoffs":[{"id":"fixture-id","driver":"fixture-driver","model":"fixture-model","title":"fixture-title","prompt":"fixture-prompt","state":"fixture-state","reply":"fixture-reply","steps":[{"kind":"fixture-kind","call_id":"fixture-call_id","tool":"fixture-tool","summary":"fixture-summary","detail":"fixture-detail","at":"2026-01-01T00:00:00Z"}]}],"via":"fixture-via","author_name":"fixture-author_name","author_avatar_url":"fixture-author_avatar_url","embeds":null},"members_only":false},
  "computer.harness_switched": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","from_kind":"fixture-from_kind","to_kind":"fixture-to_kind","harness_version":"fixture-harness_version","members_only":false},
  "computer.paired": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","server_url":"fixture-server_url","harness_version":"fixture-harness_version","token_expires_at":"2026-01-01T00:00:00Z","members_only":false},
  "computer.setup_confirmed": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","provider":"fixture-provider","confirmed_at":"2026-01-01T00:00:00Z","skills":["fixture-skills"],"members_only":false},
  "computer.setup_finished": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","run_id":"fixture-run_id","confirmed":false,"providers":[{"provider":"fixture-provider","state":"running","status":"fixture-status"}],"members_only":false},
  "computer.setup_turn_activity": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","run_id":"fixture-run_id","turn_id":"fixture-turn_id","provider":"fixture-provider","status":"fixture-status","call_id":"fixture-call_id","kind":"tool_call","tool":"fixture-tool","text":"fixture-text","at":"2026-01-01T00:00:00Z","members_only":false},
  "computer.setup_turn_changed": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","run_id":"fixture-run_id","turn_id":"fixture-turn_id","provider":"fixture-provider","provider_name":"fixture-provider_name","model":"fixture-model","state":"running","status":"fixture-status","started_at":"2026-01-01T00:00:00Z","ended_at":"2026-01-01T00:00:00Z","members_only":false},
  "computer.setup_unconfirmed": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","provider":"fixture-provider","confirmed_at":"2026-01-01T00:00:00Z","skills":["fixture-skills"],"members_only":false},
  "computer.tunnel_created": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","tunnel_id":"fixture-tunnel_id","hostname":"fixture-hostname","members_only":false},
  "computer.tunnel_removed": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","tunnel_id":"fixture-tunnel_id","hostname":"fixture-hostname","members_only":false},
  "computer.tunnel_status_changed": {"computer_id":"fixture-computer_id","user_id":"fixture-user_id","tunnel":"fixture-tunnel","harness_reachable":false,"harness_version":"fixture-harness_version","members_only":false},
  "deploy.build_completed": {"id":"fixture-id","status":"fixture-status","artifacts":["fixture-artifacts"],"error":"fixture-error"},
  "deploy.build_progress": {"id":"fixture-id","step":1,"total":1,"log":"fixture-log"},
  "deploy.build_started": {"id":"fixture-id","total":1,"log":"fixture-log"},
  "deploy.cancel_requested": {"id":"fixture-id"},
  "deploy.deploy_progress": {"id":"fixture-id","phase":"fixture-phase","log":"fixture-log"},
  "deploy.log": {"id":"fixture-id","phase":"checkout","log":"fixture-log","ts":1},
  "deploy.requested": {"id":"fixture-id","kind":"build","service":"fixture-service","target":"fixture-target","image":"fixture-image","env":{},"strategy":"fixture-strategy","repo":"fixture-repo","ref":"fixture-ref","compose_path":"fixture-compose_path","network":"fixture-network","ports":["fixture-ports"],"mounts":["fixture-mounts"],"command":["fixture-command"],"dockerfile":"fixture-dockerfile","stack_slug":"fixture-stack_slug","stack_root":"fixture-stack_root","gateway_container":"fixture-gateway_container","join_networks":["fixture-join_networks"]},
  "deploy.status_changed": {"id":"fixture-id","status":"pending","error":"fixture-error","address":"fixture-address","services":[{"name":"fixture-name","container_name":"fixture-container_name","image":"fixture-image","status":"fixture-status","networks":[{"name":"fixture-name","address":"fixture-address","gateway_container":"fixture-gateway_container","join_networks":["fixture-join_networks"]}],"ports":["fixture-ports"]}]},
  "deploy.updated": {"id":"fixture-id","status":"pending","stack_id":"fixture-stack_id"},
  "dns.exposure_changed": {"exposure_id":"fixture-exposure_id","gateway_id":"fixture-gateway_id","hostname":"fixture-hostname","service":"fixture-service","port":1,"action":"created"},
  "dns.gateway_changed": {"gateway_id":"fixture-gateway_id","kind":"fixture-kind","docker_network":"fixture-docker_network","action":"created"},
  "dns.record_changed": {"zone_id":"fixture-zone_id","zone":"fixture-zone","record_id":"fixture-record_id","action":"created","type":"fixture-type","name":"fixture-name","service":"fixture-service"},
  "dns.tunnel_changed": {"tunnel_id":"fixture-tunnel_id","name":"fixture-name","action":"created","hostname":"fixture-hostname","service":"fixture-service"},
  "doc.clarification.answer_cleared": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title"},"round":1,"question_id":"fixture-question_id","question":"fixture-question","author_id":"fixture-author_id","at":"2026-01-01T00:00:00Z"},
  "doc.clarification.answer_saved": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title"},"round":1,"question_id":"fixture-question_id","question":"fixture-question","author_id":"fixture-author_id","at":"2026-01-01T00:00:00Z"},
  "doc.clarification.anything_else_saved": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title"},"round":1,"started_by":"fixture-started_by","actor_id":"fixture-actor_id","removed":false},
  "doc.clarification.closed": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title"},"round":1,"started_by":"fixture-started_by","actor_id":"fixture-actor_id","removed":false},
  "doc.clarification.round_answered": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title"},"round":1,"started_by":"fixture-started_by","actor_id":"fixture-actor_id","removed":false},
  "doc.clarification.round_ended": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title"},"round":1,"started_by":"fixture-started_by","actor_id":"fixture-actor_id","removed":false},
  "doc.clarification.round_posted": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title"},"round":1,"started_by":"fixture-started_by","actor_id":"fixture-actor_id","removed":false,"question_count":1,"no_gaps":false},
  "doc.clarification.round_started": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title"},"round":1,"started_by":"fixture-started_by","actor_id":"fixture-actor_id","removed":false},
  "doc.created": {"doc":{"id":"fixture-id","title":"fixture-title","body":"fixture-body","project_id":"fixture-project_id","folder_id":"fixture-folder_id","version":1,"archived":false,"locked":false,"created_by":"fixture-created_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"actor_id":"fixture-actor_id","mentioned_user_ids":["fixture-mentioned_user_ids"]},
  "doc.deleted": {"id":"fixture-id","title":"fixture-title","project_id":"fixture-project_id"},
  "doc.folder.created": {"folder":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","is_default":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"actor_id":"fixture-actor_id"},
  "doc.folder.deleted": {"folder":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","is_default":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"moved_to_folder_id":"fixture-moved_to_folder_id","actor_id":"fixture-actor_id"},
  "doc.folder.updated": {"folder":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","is_default":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"previous_name":"fixture-previous_name","actor_id":"fixture-actor_id"},
  "doc.moved": {"doc":{"id":"fixture-id","title":"fixture-title","body":"fixture-body","project_id":"fixture-project_id","folder_id":"fixture-folder_id","version":1,"archived":false,"locked":false,"created_by":"fixture-created_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"from_folder_id":"fixture-from_folder_id","actor_id":"fixture-actor_id"},
  "doc.settled": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title"},"first":false,"actor_id":"fixture-actor_id"},
  "doc.updated": {"doc":{"id":"fixture-id","title":"fixture-title","body":"fixture-body","project_id":"fixture-project_id","folder_id":"fixture-folder_id","version":1,"archived":false,"locked":false,"created_by":"fixture-created_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"actor_id":"fixture-actor_id","mentioned_user_ids":["fixture-mentioned_user_ids"],"lock_changed":false},
  "doc.watchers.changed": {"doc":{"id":"fixture-id","project_id":"fixture-project_id","title":"fixture-title"},"user_id":"fixture-user_id","watching":false},
  "git.branch_deleted": {"owner":"fixture-owner","repo":"fixture-repo","branch":"fixture-branch"},
  "git.pr_closed": {"owner":"fixture-owner","repo":"fixture-repo","pr":{"number":1,"title":"fixture-title","body":"fixture-body","state":"fixture-state","merged":false,"head_sha":"fixture-head_sha","base_branch":"fixture-base_branch","author":"fixture-author","linked_ticket_ids":["fixture-linked_ticket_ids"]}},
  "git.pr_comment": {"owner":"fixture-owner","repo":"fixture-repo","pr":{"number":1},"comment":{"body":"fixture-body","author":"fixture-author"}},
  "git.pr_merged": {"owner":"fixture-owner","repo":"fixture-repo","pr":{"number":1,"title":"fixture-title","body":"fixture-body","state":"fixture-state","merged":false,"head_sha":"fixture-head_sha","base_branch":"fixture-base_branch","author":"fixture-author","linked_ticket_ids":["fixture-linked_ticket_ids"]}},
  "git.pr_opened": {"owner":"fixture-owner","repo":"fixture-repo","pr":{"number":1,"title":"fixture-title","body":"fixture-body","state":"fixture-state","merged":false,"head_sha":"fixture-head_sha","base_branch":"fixture-base_branch","author":"fixture-author","linked_ticket_ids":["fixture-linked_ticket_ids"]}},
  "git.pr_review_submitted": {"owner":"fixture-owner","repo":"fixture-repo","pr":{"number":1,"state":"fixture-state","reviewer":"fixture-reviewer"}},
  "git.provider_event": {"provider":"fixture-provider","event_type":"fixture-event_type","delivery_id":"fixture-delivery_id","action":"fixture-action","repository":{"id":1,"name":"fixture-name","full_name":"fixture-full_name","owner":"fixture-owner","html_url":"fixture-html_url","default_branch":"fixture-default_branch"},"payload":{},"received_at":"2026-01-01T00:00:00Z"},
  "git.push": {"owner":"fixture-owner","repo":"fixture-repo","branch":"fixture-branch","sha":"fixture-sha","pusher":"fixture-pusher"},
  "identity.linked": {"user_id":"fixture-user_id","provider":"fixture-provider","login":"fixture-login"},
  "identity.unlinked": {"user_id":"fixture-user_id","provider":"fixture-provider","login":"fixture-login"},
  "instance.upgrade_changed": {"id":"fixture-id","from_version":"fixture-from_version","to_version":"fixture-to_version","status":"pending","error":"fixture-error","requested_by":"fixture-requested_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},
  "instance.upgrade_requested": {"id":"fixture-id","version":"fixture-version"},
  "instance_template.updated": {"kind":"interview","key":"fixture-key","author_id":"fixture-author_id","updated_at":"2026-01-01T00:00:00Z","reset":false},
  "interview_answer.cleared": {"workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","round":1,"question":"fixture-question","author_id":"fixture-author_id","at":"2026-01-01T00:00:00Z"},
  "interview_answer.saved": {"workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","round":1,"question":"fixture-question","author_id":"fixture-author_id","at":"2026-01-01T00:00:00Z"},
  "interview_draft.dismissed": {"workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","draft_id":"fixture-draft_id","question":"fixture-question","author_id":"fixture-author_id","at":"2026-01-01T00:00:00Z"},
  "interview_draft.saved": {"workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","draft_id":"fixture-draft_id","question":"fixture-question","author_id":"fixture-author_id","at":"2026-01-01T00:00:00Z"},
  "interview_source.added": {"workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","source_id":"fixture-source_id","kind":"path","stance":"follow","author_id":"fixture-author_id","at":"2026-01-01T00:00:00Z"},
  "interview_source.changed": {"workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","source_id":"fixture-source_id","kind":"path","stance":"follow","author_id":"fixture-author_id","at":"2026-01-01T00:00:00Z"},
  "interview_source.removed": {"workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","source_id":"fixture-source_id","kind":"path","stance":"follow","author_id":"fixture-author_id","at":"2026-01-01T00:00:00Z"},
  "interview_template.updated": {"workspace_id":"fixture-workspace_id","author_id":"fixture-author_id","updated_at":"2026-01-01T00:00:00Z"},
  "invitation.created": {"invitation_id":"fixture-invitation_id","actor_id":"fixture-actor_id","user_id":"fixture-user_id","workspace_id":"fixture-workspace_id","reason":"fixture-reason"},
  "invitation.deleted": {"invitation_id":"fixture-invitation_id","actor_id":"fixture-actor_id","user_id":"fixture-user_id","workspace_id":"fixture-workspace_id","reason":"fixture-reason"},
  "invitation.redeemed": {"invitation_id":"fixture-invitation_id","actor_id":"fixture-actor_id","user_id":"fixture-user_id","workspace_id":"fixture-workspace_id","reason":"fixture-reason"},
  "invitation.revoked": {"invitation_id":"fixture-invitation_id","actor_id":"fixture-actor_id","user_id":"fixture-user_id","workspace_id":"fixture-workspace_id","reason":"fixture-reason"},
  "memory.created": {"memory":{"id":"fixture-id","workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","kind":"fixture-kind","title":"fixture-title","when_to_use":"fixture-when_to_use","always_included":false,"footer":false,"version":1,"updated_at":"2026-01-01T00:00:00Z"},"author_id":"fixture-author_id"},
  "memory.deleted": {"id":"fixture-id","workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","title":"fixture-title","author_id":"fixture-author_id"},
  "memory.updated": {"memory":{"id":"fixture-id","workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","kind":"fixture-kind","title":"fixture-title","when_to_use":"fixture-when_to_use","always_included":false,"footer":false,"version":1,"updated_at":"2026-01-01T00:00:00Z"},"author_id":"fixture-author_id","author_via":"fixture-author_via"},
  "notification.created": {"user_ids":["fixture-user_ids"],"workspace_id":"fixture-workspace_id","project_id":"fixture-project_id"},
  "notification.push_requested": {"notifications":[{"id":"fixture-id","user_id":"fixture-user_id","workspace_id":"fixture-workspace_id"}]},
  "personal_access_token.minted": {"token_id":"fixture-token_id","user_id":"fixture-user_id","name":"fixture-name","computer_id":"fixture-computer_id"},
  "personal_access_token.revoked": {"token_id":"fixture-token_id","user_id":"fixture-user_id","name":"fixture-name","computer_id":"fixture-computer_id"},
  "play.created": {"play":{"id":"fixture-id","workspace_id":"fixture-workspace_id","label":"fixture-label","type":"fixture-type","description":"fixture-description","instructions":"fixture-instructions","enabled":false,"show_when_stage":"fixture-show_when_stage","excluded_project_ids":["fixture-excluded_project_ids"],"builtin_key":"fixture-builtin_key","created_by":"fixture-created_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "play.deleted": {"id":"fixture-id","label":"fixture-label","workspace_id":"fixture-workspace_id"},
  "play.queue_resumed": {"workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","target_type":"ticket","target_id":"fixture-target_id","resumed_by":"fixture-resumed_by","resumed_at":"2026-01-01T00:00:00Z"},
  "play.queue_updated": {"id":"fixture-id","workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","target_type":"ticket","target_id":"fixture-target_id","play_id":"fixture-play_id","play_label":"fixture-play_label","auto_play_id":"fixture-auto_play_id","automation_id":"fixture-automation_id","person_id":"fixture-person_id","run_on":"developer","moment":"ticket.unblocked","priority":"high","status":"queued","reason":"fixture-reason","trail_id":"fixture-trail_id","via":"web","queued_at":"2026-01-01T00:00:00Z","decided_at":"2026-01-01T00:00:00Z","not_before":"2026-01-01T00:00:00Z"},
  "play.queued": {"id":"fixture-id","workspace_id":"fixture-workspace_id","project_id":"fixture-project_id","target_type":"ticket","target_id":"fixture-target_id","play_id":"fixture-play_id","play_label":"fixture-play_label","auto_play_id":"fixture-auto_play_id","automation_id":"fixture-automation_id","person_id":"fixture-person_id","run_on":"developer","moment":"ticket.unblocked","priority":"high","status":"queued","reason":"fixture-reason","trail_id":"fixture-trail_id","via":"web","queued_at":"2026-01-01T00:00:00Z","decided_at":"2026-01-01T00:00:00Z","not_before":"2026-01-01T00:00:00Z"},
  "play.run_finished": {"trail_id":"fixture-trail_id","play_id":"fixture-play_id","play_label":"fixture-play_label","target_type":"ticket","target_id":"fixture-target_id","target_title":"fixture-target_title","starter_id":"fixture-starter_id","via":"web","workspace_id":"fixture-workspace_id","outcome":"done","last_error":"fixture-last_error","reply_message_id":"fixture-reply_message_id"},
  "play.run_started": {"trail_id":"fixture-trail_id","play_id":"fixture-play_id","play_label":"fixture-play_label","target_type":"ticket","target_id":"fixture-target_id","target_title":"fixture-target_title","starter_id":"fixture-starter_id","via":"web","workspace_id":"fixture-workspace_id","harness_session_id":"fixture-harness_session_id"},
  "play.run_waiting": {"trail_id":"fixture-trail_id","play_id":"fixture-play_id","play_label":"fixture-play_label","target_type":"ticket","target_id":"fixture-target_id","target_title":"fixture-target_title","starter_id":"fixture-starter_id","via":"web","workspace_id":"fixture-workspace_id"},
  "play.updated": {"play":{"id":"fixture-id","workspace_id":"fixture-workspace_id","label":"fixture-label","type":"fixture-type","description":"fixture-description","instructions":"fixture-instructions","enabled":false,"show_when_stage":"fixture-show_when_stage","excluded_project_ids":["fixture-excluded_project_ids"],"builtin_key":"fixture-builtin_key","created_by":"fixture-created_by","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "project.setup_changed": {"project_id":"fixture-project_id","workspace_id":"fixture-workspace_id","setup":{"stack_id":"fixture-stack_id","env_keys":["fixture-env_keys"],"finished":false,"steps":{}}},
  "repository.installation.assigned": {"account_id":1,"account_login":"fixture-account_login","workspace_id":"fixture-workspace_id","actor_id":"fixture-actor_id","uninstalled":false},
  "repository.installation.unassigned": {"account_id":1,"account_login":"fixture-account_login","workspace_id":"fixture-workspace_id","actor_id":"fixture-actor_id","uninstalled":false},
  "review.status_changed": {"id":"fixture-id","repo":"fixture-repo","pr_number":1,"status":"fixture-status","reviewer":"fixture-reviewer"},
  "role.updated": {"role_id":"fixture-role_id","workspace_id":"fixture-workspace_id","actor_id":"fixture-actor_id"},
  "runner.connected": {"runner_id":"fixture-runner_id","name":"fixture-name"},
  "runner.disconnected": {"runner_id":"fixture-runner_id","reason":"fixture-reason"},
  "runner.facts_reported": {"runner_id":"fixture-runner_id","computer_id":"fixture-computer_id","user_id":"fixture-user_id","facts":{"hostname":"fixture-hostname","t3":{"state":"answering","port":1,"version":"fixture-version"}},"members_only":false},
  "runner.heartbeat": {"runner_id":"fixture-runner_id","ts":1},
  "runner.personal_changed": {"runner_id":"fixture-runner_id","computer_id":"fixture-computer_id","user_id":"fixture-user_id","state":"enrolled","hostname":"fixture-hostname","members_only":false},
  "service.created": {"stack":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","slug":"fixture-slug","machine":"fixture-machine","strategy":"fixture-strategy","compose_path":"fixture-compose_path","env":{},"docker_network":"fixture-docker_network","ports":["fixture-ports"],"mounts":["fixture-mounts"],"command":["fixture-command"],"build_source":{"repo_owner":"fixture-repo_owner","repo_name":"fixture-repo_name","branch":"fixture-branch","dockerfile":"fixture-dockerfile","compose_path":"fixture-compose_path"},"branch_deploy_rules":[{"pattern":"fixture-pattern","docker_network":"fixture-docker_network","hostname_template":"fixture-hostname_template","name_suffix":"fixture-name_suffix","port":1,"overrides":{}}],"derived_from":"fixture-derived_from","branch":"fixture-branch","managed":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"services":[{"id":"fixture-id","stack_id":"fixture-stack_id","name":"fixture-name","declared":{"image":"fixture-image","build":"fixture-build","ports":["fixture-ports"],"env_keys":["fixture-env_keys"]},"container_name":"fixture-container_name","image":"fixture-image","status":"fixture-status","networks":[{"name":"fixture-name","address":"fixture-address"}],"ports":["fixture-ports"],"observed_at":"2026-01-01T00:00:00Z"}]},
  "service.deleted": {"id":"fixture-id","name":"fixture-name","service_ids":["fixture-service_ids"],"project_id":"fixture-project_id"},
  "service.updated": {"stack":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","slug":"fixture-slug","machine":"fixture-machine","strategy":"fixture-strategy","compose_path":"fixture-compose_path","env":{},"docker_network":"fixture-docker_network","ports":["fixture-ports"],"mounts":["fixture-mounts"],"command":["fixture-command"],"build_source":{"repo_owner":"fixture-repo_owner","repo_name":"fixture-repo_name","branch":"fixture-branch","dockerfile":"fixture-dockerfile","compose_path":"fixture-compose_path"},"branch_deploy_rules":[{"pattern":"fixture-pattern","docker_network":"fixture-docker_network","hostname_template":"fixture-hostname_template","name_suffix":"fixture-name_suffix","port":1,"overrides":{}}],"derived_from":"fixture-derived_from","branch":"fixture-branch","managed":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"services":[{"id":"fixture-id","stack_id":"fixture-stack_id","name":"fixture-name","declared":{"image":"fixture-image","build":"fixture-build","ports":["fixture-ports"],"env_keys":["fixture-env_keys"]},"container_name":"fixture-container_name","image":"fixture-image","status":"fixture-status","networks":[{"name":"fixture-name","address":"fixture-address"}],"ports":["fixture-ports"],"observed_at":"2026-01-01T00:00:00Z"}]},
  "session.created": {"session_id":"fixture-session_id","user_id":"fixture-user_id","client":"browser","platform":"fixture-platform","label":"fixture-label"},
  "session.revoked": {"session_id":"fixture-session_id","user_id":"fixture-user_id","client":"browser","platform":"fixture-platform","label":"fixture-label"},
  "status.created": {"status":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","position":1,"kind":"fixture-kind","icon":"fixture-icon","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"previous_kind":"fixture-previous_kind"},
  "status.deleted": {"status":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","position":1,"kind":"fixture-kind","icon":"fixture-icon","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"previous_kind":"fixture-previous_kind"},
  "status.updated": {"status":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","position":1,"kind":"fixture-kind","icon":"fixture-icon","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},"previous_kind":"fixture-previous_kind"},
  "ticket.assignee_changed": {"ticket":{"id":"fixture-id","project_id":"fixture-project_id","category_id":"fixture-category_id","type_id":"fixture-type_id","title":"fixture-title","body":"fixture-body","status":"fixture-status","position":1,"number":1,"doc_id":"fixture-doc_id","developer":"fixture-developer","tester":"fixture-tester","reporter":{"kind":"user","login":"fixture-login","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:00Z","labels":["fixture-labels"],"assignee":"fixture-assignee"},"from":"fixture-from","to":"fixture-to"},
  "ticket.category_changed": {"ticket_id":"fixture-ticket_id","category_id":"fixture-category_id","project_id":"fixture-project_id"},
  "ticket.created": {"ticket":{"id":"fixture-id","project_id":"fixture-project_id","category_id":"fixture-category_id","type_id":"fixture-type_id","title":"fixture-title","body":"fixture-body","status":"fixture-status","position":1,"number":1,"doc_id":"fixture-doc_id","developer":"fixture-developer","tester":"fixture-tester","reporter":{"kind":"user","login":"fixture-login","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:00Z","labels":["fixture-labels"],"assignee":"fixture-assignee"},"mentioned_user_ids":["fixture-mentioned_user_ids"]},
  "ticket.deleted": {"id":"fixture-id","title":"fixture-title","project_id":"fixture-project_id"},
  "ticket.developer_changed": {"ticket":{"id":"fixture-id","project_id":"fixture-project_id","category_id":"fixture-category_id","type_id":"fixture-type_id","title":"fixture-title","body":"fixture-body","status":"fixture-status","position":1,"number":1,"doc_id":"fixture-doc_id","developer":"fixture-developer","tester":"fixture-tester","reporter":{"kind":"user","login":"fixture-login","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:00Z","labels":["fixture-labels"],"assignee":"fixture-assignee"},"from":"fixture-from","to":"fixture-to","actor":{"kind":"user","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name","play_label":"fixture-play_label","trail_id":"fixture-trail_id","user_id":"fixture-user_id"}},
  "ticket.finished": {"ticket":{"id":"fixture-id","project_id":"fixture-project_id","category_id":"fixture-category_id","type_id":"fixture-type_id","title":"fixture-title","body":"fixture-body","status":"fixture-status","position":1,"number":1,"doc_id":"fixture-doc_id","developer":"fixture-developer","tester":"fixture-tester","reporter":{"kind":"user","login":"fixture-login","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:00Z","labels":["fixture-labels"],"assignee":"fixture-assignee"}},
  "ticket.link_created": {"link":{"ticket_id":"fixture-ticket_id","kind":"found_in","target_id":"fixture-target_id","created_at":"2026-01-01T00:00:00Z"}},
  "ticket.link_deleted": {"link":{"ticket_id":"fixture-ticket_id","kind":"found_in","target_id":"fixture-target_id","created_at":"2026-01-01T00:00:00Z"}},
  "ticket.status_changed": {"ticket":{"id":"fixture-id","project_id":"fixture-project_id","category_id":"fixture-category_id","type_id":"fixture-type_id","title":"fixture-title","body":"fixture-body","status":"fixture-status","position":1,"number":1,"doc_id":"fixture-doc_id","developer":"fixture-developer","tester":"fixture-tester","reporter":{"kind":"user","login":"fixture-login","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:00Z","labels":["fixture-labels"],"assignee":"fixture-assignee"},"from":"fixture-from","to":"fixture-to","actor":{"kind":"user","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name","play_label":"fixture-play_label","trail_id":"fixture-trail_id","user_id":"fixture-user_id"},"run_id":"fixture-run_id"},
  "ticket.test_failed": {"ticket":{"id":"fixture-id","project_id":"fixture-project_id","category_id":"fixture-category_id","type_id":"fixture-type_id","title":"fixture-title","body":"fixture-body","status":"fixture-status","position":1,"number":1,"doc_id":"fixture-doc_id","developer":"fixture-developer","tester":"fixture-tester","reporter":{"kind":"user","login":"fixture-login","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:00Z","labels":["fixture-labels"],"assignee":"fixture-assignee"},"tester":"fixture-tester","report":"fixture-report"},
  "ticket.test_passed": {"ticket":{"id":"fixture-id","project_id":"fixture-project_id","category_id":"fixture-category_id","type_id":"fixture-type_id","title":"fixture-title","body":"fixture-body","status":"fixture-status","position":1,"number":1,"doc_id":"fixture-doc_id","developer":"fixture-developer","tester":"fixture-tester","reporter":{"kind":"user","login":"fixture-login","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:00Z","labels":["fixture-labels"],"assignee":"fixture-assignee"},"tester":"fixture-tester","report":"fixture-report"},
  "ticket.tester_changed": {"ticket":{"id":"fixture-id","project_id":"fixture-project_id","category_id":"fixture-category_id","type_id":"fixture-type_id","title":"fixture-title","body":"fixture-body","status":"fixture-status","position":1,"number":1,"doc_id":"fixture-doc_id","developer":"fixture-developer","tester":"fixture-tester","reporter":{"kind":"user","login":"fixture-login","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:00Z","labels":["fixture-labels"],"assignee":"fixture-assignee"},"from":"fixture-from","to":"fixture-to","actor":{"kind":"user","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name","play_label":"fixture-play_label","trail_id":"fixture-trail_id","user_id":"fixture-user_id"}},
  "ticket.unblocked": {"ticket_id":"fixture-ticket_id","project_id":"fixture-project_id","blocker_id":"fixture-blocker_id","cause":"blocker_done","actor":{"kind":"user","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name","play_label":"fixture-play_label","trail_id":"fixture-trail_id","user_id":"fixture-user_id"}},
  "ticket.updated": {"ticket":{"id":"fixture-id","project_id":"fixture-project_id","category_id":"fixture-category_id","type_id":"fixture-type_id","title":"fixture-title","body":"fixture-body","status":"fixture-status","position":1,"number":1,"doc_id":"fixture-doc_id","developer":"fixture-developer","tester":"fixture-tester","reporter":{"kind":"user","login":"fixture-login","automation_id":"fixture-automation_id","automation_name":"fixture-automation_name"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","finished_at":"2026-01-01T00:00:00Z","labels":["fixture-labels"],"assignee":"fixture-assignee"},"actor_id":"fixture-actor_id","mentioned_user_ids":["fixture-mentioned_user_ids"]},
  "ticket_type.created": {"ticket_type":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","position":1,"color":"fixture-color","body_template":"fixture-body_template","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "ticket_type.deleted": {"ticket_type":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","position":1,"color":"fixture-color","body_template":"fixture-body_template","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "ticket_type.updated": {"ticket_type":{"id":"fixture-id","project_id":"fixture-project_id","name":"fixture-name","position":1,"color":"fixture-color","body_template":"fixture-body_template","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}},
  "topology.updated": {"environment":"fixture-environment","workspace_id":"fixture-workspace_id","canvas":{"schema_version":1,"nodes":[{"id":"fixture-id","type":"fixture-type","position":{"x":1,"y":1},"data":{"service_id":"fixture-service_id","name":"fixture-name","runtime":"fixture-runtime","url":"fixture-url","status":"fixture-status","replicas":1,"volume":"fixture-volume","address":"fixture-address","label":"fixture-label"}}],"edges":[{"id":"fixture-id","source":"fixture-source","target":"fixture-target","type":"fixture-type","data":{"kind":"fixture-kind"}}],"viewport":{"x":1,"y":1,"zoom":1}}},
  "voice.occupancy.changed": {"conversation_id":"fixture-conversation_id","occupants":[{"identity":"fixture-identity","name":"fixture-name"}],"members_only":false},
  "workspace.member.added": {"invitation_id":"fixture-invitation_id","actor_id":"fixture-actor_id","user_id":"fixture-user_id","workspace_id":"fixture-workspace_id","reason":"fixture-reason","project_ids":["fixture-project_ids"]},
  "workspace.member.removed": {"user_id":"fixture-user_id","workspace_id":"fixture-workspace_id","actor_id":"fixture-actor_id","project_ids":["fixture-project_ids"]},
  "workspace.member.updated": {"user_id":"fixture-user_id","workspace_id":"fixture-workspace_id","actor_id":"fixture-actor_id","project_ids":["fixture-project_ids"]},
  "workspace.updated": {"workspace_id":"fixture-workspace_id","name":"fixture-name","slug":"fixture-slug","actor_id":"fixture-actor_id"},
};
