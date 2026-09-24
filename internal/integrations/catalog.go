package integrations

import "context"

// This file defines the published event-schema catalog (ADR 0044), seeded into event_schemas on startup.

// catalogSchemas maps topic -> latest published schema JSON; PublishCatalog assigns version numbers from map position.
var catalogSchemas = map[string]string{
	"invitation.created":     `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"invitation_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"invitation.revoked":     `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"invitation_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"invitation.redeemed":    `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"invitation_id":{"type":"string"},"user_id":{"type":"string"}}}`,
	"invitation.deleted":     `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"invitation_id":{"type":"string"},"reason":{"type":"string"}}}`,
	"account.admitted":       `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"invitation_id":{"type":"string"},"user_id":{"type":"string"}}}`,
	"account.disabled":       `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["account_id"],"properties":{"account_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"account.reactivated":    `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["account_id"],"properties":{"account_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"account.removed":        `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["account_id"],"properties":{"account_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"account.restored":       `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["account_id"],"properties":{"account_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"workspace.member.added": `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"invitation_id":{"type":"string"},"user_id":{"type":"string"},"workspace_id":{"type":"string"}}}`,
	"doc.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["doc"],
		"properties": {
			"doc": {
				"type": "object",
				"required": ["id", "title", "version"],
				"properties": {
					"id": {"type": "string"},
					"title": {"type": "string"},
					"body": {"type": "string"},
					"version": {"type": "integer"},
					"archived": {"type": "boolean"},
					"created_at": {"type": "string", "format": "date-time"},
					"updated_at": {"type": "string", "format": "date-time"}
				}
			}
		}
	}`,
	"doc.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["doc"],
		"properties": {
			"doc": {
				"type": "object",
				"required": ["id", "title", "version"],
				"properties": {
					"id": {"type": "string"},
					"title": {"type": "string"},
					"body": {"type": "string"},
					"version": {"type": "integer"},
					"archived": {"type": "boolean"},
					"created_at": {"type": "string", "format": "date-time"},
					"updated_at": {"type": "string", "format": "date-time"}
				}
			}
		}
	}`,
	"ticket.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["ticket"],
		"properties": {
			"ticket": {
				"type": "object",
				"required": ["id", "title", "status"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"title": {"type": "string"},
					"body": {"type": "string"},
					"status": {"type": "string", "enum": ["open", "in_progress", "done", "closed"]},
					"doc_id": {"type": "string"},
					"assignee": {"type": "string"},
					"created_at": {"type": "string", "format": "date-time"},
					"updated_at": {"type": "string", "format": "date-time"},
					"finished_at": {"type": ["string", "null"], "format": "date-time"}
				}
			}
		}
	}`,
	"ticket.status_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["ticket", "from", "to"],
		"properties": {
			"ticket": {"type": "object"},
			"from": {"type": "string"},
			"to": {"type": "string"},
			"actor": {
				"type": "object",
				"properties": {
					"kind": {"type": "string", "enum": ["user", "automation", "play", "play:mcp"]},
					"automation_id": {"type": "string"},
					"automation_name": {"type": "string"},
					"play_label": {"type": "string"},
					"trail_id": {"type": "string"}
				}
			},
			"execution_id": {"type": "string"}
		}
	}`,
	"ticket.finished": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["ticket"],
		"properties": {"ticket": {"type": "object"}}
	}`,
	// deploy.requested has no "steps" field, that belongs to the runner's own consumer type; this matches the wire shape.
	"deploy.requested": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "kind"],
		"properties": {
			"id": {"type": "string"},
			"kind": {"type": "string", "enum": ["build", "deploy"]},
			"service": {"type": "string"},
			"target": {"type": "string"},
			"image": {"type": "string"},
			"env": {
				"type": "object",
				"additionalProperties": {"type": "string"},
				"description": "Deploy environment keys; values are always empty strings. The deploy domain redacts them before publish (ticket 14), so every consumer of this event — not just webhook delivery — only ever sees keys."
			},
			"strategy": {"type": "string"},
			"repo": {"type": "string"},
			"ref": {"type": "string"},
			"compose_dir": {"type": "string"},
			"network": {"type": "string"},
			"ports": {"type": "array", "items": {"type": "string"}},
			"mounts": {"type": "array", "items": {"type": "string"}},
			"dockerfile": {"type": "string"},
			"compose_path": {"type": "string"},
			"health_check": {"type": "object"}
		}
	}`,
	"deploy.status_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "status"],
		"properties": {
			"id": {"type": "string"},
			"status": {"type": "string", "enum": ["pending", "running", "healthy", "failed"]},
			"error": {"type": "string"}
		}
	}`,
	"deploy.cancel_requested": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id"],
		"properties": {"id": {"type": "string"}}
	}`,
	"topology.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["environment"],
		"properties": {
			"environment": {"type": "string"},
			"canvas": {"type": "object"}
		}
	}`,
	"git.provider_event": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["provider", "event_type", "delivery_id", "received_at"],
		"properties": {
			"provider": {"type": "string"},
			"event_type": {"type": "string"},
			"delivery_id": {"type": "string"},
			"action": {"type": "string"},
			"repository": {"type": "object"},
			"payload": {"type": "object"},
			"received_at": {"type": "string", "format": "date-time"}
		}
	}`,
	"git.pr_opened": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["owner", "repo", "pr"],
		"properties": {
			"owner": {"type": "string"},
			"repo": {"type": "string"},
			"pr": {"type": "object"}
		}
	}`,
	"git.pr_merged": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["owner", "repo", "pr"],
		"properties": {
			"owner": {"type": "string"},
			"repo": {"type": "string"},
			"pr": {"type": "object"}
		}
	}`,
	"git.pr_closed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["owner", "repo", "pr"],
		"properties": {
			"owner": {"type": "string"},
			"repo": {"type": "string"},
			"pr": {"type": "object"}
		}
	}`,
	"git.pr_review_submitted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["owner", "repo", "pr"],
		"properties": {
			"owner": {"type": "string"},
			"repo": {"type": "string"},
			"pr": {"type": "object"}
		}
	}`,
	"review.status_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "repo", "pr_number", "status"],
		"properties": {
			"id": {"type": "string"},
			"repo": {"type": "string"},
			"pr_number": {"type": "integer"},
			"status": {"type": "string"},
			"reviewer": {"type": "string"}
		}
	}`,
	// notification.created's real payload (workspace.NotificationCreatedEvent) is
	// intentionally empty — the topic alone carries the meaning.
	"notification.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {}
	}`,
	"runner.connected": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["runner_id"],
		"properties": {
			"runner_id": {"type": "string"},
			"name": {"type": "string"}
		}
	}`,
	"runner.disconnected": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["runner_id"],
		"properties": {
			"runner_id": {"type": "string"},
			"reason": {"type": "string"}
		}
	}`,
	"runner.heartbeat": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["runner_id", "ts"],
		"properties": {
			"runner_id": {"type": "string"},
			"ts": {"type": "integer"}
		}
	}`,
	"instance.upgrade_requested": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "version"],
		"properties": {
			"id": {"type": "string"},
			"version": {"type": "string"}
		}
	}`,
	"instance.upgrade_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "from_version", "to_version", "status"],
		"properties": {
			"id": {"type": "string"},
			"from_version": {"type": "string"},
			"to_version": {"type": "string"},
			"status": {"type": "string", "enum": ["pending", "started", "completed", "failed"]},
			"error": {"type": "string"},
			"requested_by": {"type": "string"},
			"created_at": {"type": "string", "format": "date-time"},
			"updated_at": {"type": "string", "format": "date-time"}
		}
	}`,
	"deploy.build_started": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "total"],
		"properties": {
			"id": {"type": "string"},
			"total": {"type": "integer"},
			"log": {"type": "string"}
		}
	}`,
	"deploy.build_progress": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "step", "total"],
		"properties": {
			"id": {"type": "string"},
			"step": {"type": "integer"},
			"total": {"type": "integer"},
			"log": {"type": "string"}
		}
	}`,
	"deploy.build_completed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "status"],
		"properties": {
			"id": {"type": "string"},
			"status": {"type": "string"},
			"artifacts": {"type": "array", "items": {"type": "string"}},
			"error": {"type": "string"}
		}
	}`,
	"deploy.deploy_progress": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "phase"],
		"properties": {
			"id": {"type": "string"},
			"phase": {"type": "string"},
			"log": {"type": "string"}
		}
	}`,
	"deploy.log": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "phase", "log", "ts"],
		"properties": {
			"id": {"type": "string"},
			"phase": {"type": "string", "enum": ["checkout", "build", "deploy"]},
			"log": {"type": "string"},
			"ts": {"type": "integer"}
		}
	}`,
	"deploy.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "status"],
		"properties": {
			"id": {"type": "string"},
			"status": {"type": "string", "enum": ["pending", "running", "healthy", "failed"]}
		}
	}`,
	"dns.record_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["zone_id", "action"],
		"properties": {
			"zone_id": {"type": "string"},
			"zone": {"type": "string"},
			"record_id": {"type": "string"},
			"action": {"type": "string", "enum": ["created", "updated", "deleted"]},
			"type": {"type": "string"},
			"name": {"type": "string"},
			"service": {"type": "string"}
		}
	}`,
	"dns.tunnel_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["tunnel_id", "action"],
		"properties": {
			"tunnel_id": {"type": "string"},
			"name": {"type": "string"},
			"action": {"type": "string", "enum": ["created", "routed", "rotated", "deleted"]},
			"hostname": {"type": "string"},
			"service": {"type": "string"}
		}
	}`,
	"dns.gateway_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["gateway_id", "action"],
		"properties": {
			"gateway_id": {"type": "string"},
			"kind": {"type": "string"},
			"docker_network": {"type": "string"},
			"action": {"type": "string", "enum": ["created", "deleted"]}
		}
	}`,
	"dns.exposure_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["exposure_id", "gateway_id", "action"],
		"properties": {
			"exposure_id": {"type": "string"},
			"gateway_id": {"type": "string"},
			"hostname": {"type": "string"},
			"service": {"type": "string"},
			"port": {"type": "integer"},
			"action": {"type": "string", "enum": ["created", "deleted"]}
		}
	}`,
	"service.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["service"],
		"properties": {"service": {"type": "object"}}
	}`,
	"service.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["service"],
		"properties": {"service": {"type": "object"}}
	}`,
	"service.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "name"],
		"properties": {
			"id": {"type": "string"},
			"name": {"type": "string"}
		}
	}`,
	"git.push": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["owner", "repo", "branch", "sha"],
		"properties": {
			"owner": {"type": "string"},
			"repo": {"type": "string"},
			"branch": {"type": "string"},
			"sha": {"type": "string"},
			"pusher": {"type": "string"}
		}
	}`,
	"git.branch_deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["owner", "repo", "branch"],
		"properties": {
			"owner": {"type": "string"},
			"repo": {"type": "string"},
			"branch": {"type": "string"}
		}
	}`,
	"git.pr_comment": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["owner", "repo", "pr", "comment"],
		"properties": {
			"owner": {"type": "string"},
			"repo": {"type": "string"},
			"pr": {
				"type": "object",
				"required": ["number"],
				"properties": {"number": {"type": "integer"}}
			},
			"comment": {
				"type": "object",
				"required": ["body", "author"],
				"properties": {
					"body": {"type": "string"},
					"author": {"type": "string"}
				}
			}
		}
	}`,
	"ticket.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["ticket"],
		"properties": {"ticket": {"type": "object"}}
	}`,
	"ticket.assignee_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["ticket", "from", "to"],
		"properties": {
			"ticket": {"type": "object"},
			"from": {"type": "string"},
			"to": {"type": "string"}
		}
	}`,
	"ticket.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "title"],
		"properties": {
			"id": {"type": "string"},
			"title": {"type": "string"}
		}
	}`,
	"doc.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "title"],
		"properties": {
			"id": {"type": "string"},
			"title": {"type": "string"}
		}
	}`,
	"play.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["play"],
		"properties": {
			"play": {
				"type": "object",
				"required": ["id", "workspace_id", "label", "type"],
				"properties": {
					"id": {"type": "string"},
					"workspace_id": {"type": "string"},
					"label": {"type": "string"},
					"type": {"type": "string"},
					"description": {"type": "string"},
					"instructions": {"type": "string"},
					"enabled": {"type": "boolean"},
					"show_when_stage": {"type": ["string", "null"]},
					"excluded_project_ids": {"type": "array", "items": {"type": "string"}},
					"created_by": {"type": "string"},
					"created_at": {"type": "string", "format": "date-time"},
					"updated_at": {"type": "string", "format": "date-time"}
				}
			}
		}
	}`,
	"play.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["play"],
		"properties": {
			"play": {
				"type": "object",
				"required": ["id", "workspace_id", "label", "type"],
				"properties": {
					"id": {"type": "string"},
					"workspace_id": {"type": "string"},
					"label": {"type": "string"},
					"type": {"type": "string"},
					"description": {"type": "string"},
					"instructions": {"type": "string"},
					"enabled": {"type": "boolean"},
					"show_when_stage": {"type": ["string", "null"]},
					"excluded_project_ids": {"type": "array", "items": {"type": "string"}},
					"created_by": {"type": "string"},
					"created_at": {"type": "string", "format": "date-time"},
					"updated_at": {"type": "string", "format": "date-time"}
				}
			}
		}
	}`,
	"play.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "label"],
		"properties": {
			"id": {"type": "string"},
			"label": {"type": "string"}
		}
	}`,
	"play.run_started": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["trail_id", "play_id", "play_label", "target_type", "target_id", "starter_id", "via"],
		"properties": {
			"trail_id": {"type": "string"},
			"play_id": {"type": "string"},
			"play_label": {"type": "string"},
			"target_type": {"type": "string", "enum": ["ticket", "doc"]},
			"target_id": {"type": "string"},
			"target_title": {"type": "string"},
			"starter_id": {"type": "string"},
			"via": {"type": "string", "enum": ["web", "mcp"]},
			"harness_session_id": {"type": "string"}
		}
	}`,
	"play.run_waiting": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["trail_id", "play_id", "play_label", "target_type", "target_id", "starter_id", "via"],
		"properties": {
			"trail_id": {"type": "string"},
			"play_id": {"type": "string"},
			"play_label": {"type": "string"},
			"target_type": {"type": "string", "enum": ["ticket", "doc"]},
			"target_id": {"type": "string"},
			"target_title": {"type": "string"},
			"starter_id": {"type": "string"},
			"via": {"type": "string", "enum": ["web", "mcp"]}
		}
	}`,
	"play.run_finished": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["trail_id", "play_id", "play_label", "target_type", "target_id", "starter_id", "via", "outcome"],
		"properties": {
			"trail_id": {"type": "string"},
			"play_id": {"type": "string"},
			"play_label": {"type": "string"},
			"target_type": {"type": "string", "enum": ["ticket", "doc"]},
			"target_id": {"type": "string"},
			"target_title": {"type": "string"},
			"starter_id": {"type": "string"},
			"via": {"type": "string", "enum": ["web", "mcp"]},
			"outcome": {"type": "string", "enum": ["done", "failed", "interrupted"]},
			"last_error": {"type": "string"},
			"reply_message_id": {"type": "string"}
		}
	}`,
	"memory.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["memory", "author_id"],
		"properties": {
			"memory": {
				"type": "object",
				"required": ["id", "workspace_id", "project_id", "title"],
				"properties": {
					"id": {"type": "string"},
					"workspace_id": {"type": "string"},
					"project_id": {"type": "string"},
					"title": {"type": "string"},
					"when_to_use": {"type": "string"},
					"always_included": {"type": "boolean"},
					"updated_at": {"type": "string", "format": "date-time"}
				}
			},
			"author_id": {"type": "string"}
		}
	}`,
	"memory.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["memory", "author_id"],
		"properties": {
			"memory": {
				"type": "object",
				"required": ["id", "workspace_id", "project_id", "title"],
				"properties": {
					"id": {"type": "string"},
					"workspace_id": {"type": "string"},
					"project_id": {"type": "string"},
					"title": {"type": "string"},
					"when_to_use": {"type": "string"},
					"always_included": {"type": "boolean"},
					"updated_at": {"type": "string", "format": "date-time"}
				}
			},
			"author_id": {"type": "string"}
		}
	}`,
	"memory.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "title", "author_id"],
		"properties": {
			"id": {"type": "string"},
			"title": {"type": "string"},
			"author_id": {"type": "string"}
		}
	}`,
	"computer.setup_confirmed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["computer_id", "user_id"],
		"properties": {
			"computer_id": {"type": "string"},
			"user_id": {"type": "string"},
			"provider": {"type": "string"},
			"confirmed_at": {"type": "string", "format": "date-time"},
			"skills": {"type": "array", "items": {"type": "string"}}
		}
	}`,
	"computer.setup_unconfirmed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["computer_id", "user_id"],
		"properties": {
			"computer_id": {"type": "string"},
			"user_id": {"type": "string"},
			"provider": {"type": "string"},
			"confirmed_at": {"type": "string", "format": "date-time"},
			"skills": {"type": "array", "items": {"type": "string"}}
		}
	}`,
	"ticket.category_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["ticket_id", "category_id"],
		"properties": {
			"ticket_id": {"type": "string"},
			"category_id": {"type": "string"}
		}
	}`,
	"category.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["category"],
		"properties": {"category": {"type": "object"}}
	}`,
	"category.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["category"],
		"properties": {"category": {"type": "object"}}
	}`,
	"category.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["category"],
		"properties": {"category": {"type": "object"}}
	}`,
	"ticket_type.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["ticket_type"],
		"properties": {"ticket_type": {"type": "object"}}
	}`,
	"ticket_type.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["ticket_type"],
		"properties": {"ticket_type": {"type": "object"}}
	}`,
	"ticket_type.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["ticket_type"],
		"properties": {"ticket_type": {"type": "object"}}
	}`,
	"status.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["status"],
		"properties": {"status": {"type": "object"}}
	}`,
	"status.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["status"],
		"properties": {"status": {"type": "object"}}
	}`,
	"status.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["status"],
		"properties": {"status": {"type": "object"}}
	}`,
	"chat.conversation.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["conversation"],
		"properties": {"conversation": {"type": "object"}}
	}`,
	"chat.message.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["message"],
		"properties": {"message": {"type": "object"}}
	}`,
	"chat.message.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["message"],
		"properties": {"message": {"type": "object"}}
	}`,
	"chat.message.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["conversation_id", "message_id", "deleted_at"],
		"properties": {
			"conversation_id": {"type": "string"},
			"message_id": {"type": "string"},
			"deleted_at": {"type": "string", "format": "date-time"}
		}
	}`,
	"voice.occupancy.changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["conversation_id", "occupants"],
		"properties": {
			"conversation_id": {"type": "string"},
			"occupants": {
				"type": "array",
				"items": {
					"type": "object",
					"required": ["identity", "name"],
					"properties": {
						"identity": {"type": "string"},
						"name": {"type": "string"}
					}
				}
			}
		}
	}`,
}

// PublishCatalog seeds catalog schemas idempotently before any integration installs.
func (s *Service) PublishCatalog(ctx context.Context) error {
	for topic, schema := range catalogSchemas {
		if err := s.cfg.Schemas.Publish(ctx, SchemaEntry{
			Topic:     topic,
			Version:   1,
			Schema:    schema,
			CreatedAt: s.cfg.Now().UTC(),
		}); err != nil {
			return err
		}
	}
	return nil
}
