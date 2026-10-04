package integrations

import "context"

// This file defines the published event-schema catalog (ADR 0044), seeded into event_schemas on startup.

// catalogSchemas maps topic -> latest published schema JSON; PublishCatalog assigns version numbers from map position.
var catalogSchemas = map[string]string{
	"invitation.created":       `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"invitation_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"invitation.revoked":       `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"invitation_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"invitation.redeemed":      `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"invitation_id":{"type":"string"},"user_id":{"type":"string"}}}`,
	"invitation.deleted":       `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"invitation_id":{"type":"string"},"reason":{"type":"string"}}}`,
	"account.admitted":         `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"invitation_id":{"type":"string"},"user_id":{"type":"string"}}}`,
	"account.disabled":         `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["account_id"],"properties":{"account_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"account.reactivated":      `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["account_id"],"properties":{"account_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"account.removed":          `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["account_id"],"properties":{"account_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"account.restored":         `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["account_id"],"properties":{"account_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"account.profile_updated":  `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["account_id"],"properties":{"account_id":{"type":"string"}}}`,
	"workspace.member.added":   `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"invitation_id":{"type":"string"},"user_id":{"type":"string"},"workspace_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"workspace.member.removed": `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["user_id","workspace_id"],"properties":{"user_id":{"type":"string"},"workspace_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"workspace.member.updated": `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["user_id","workspace_id"],"properties":{"user_id":{"type":"string"},"workspace_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"workspace.updated":        `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["workspace_id","name","slug"],"properties":{"workspace_id":{"type":"string"},"name":{"type":"string"},"slug":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"role.updated":             `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["role_id","workspace_id"],"properties":{"role_id":{"type":"string"},"workspace_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
	"access.grant.changed":     `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["resource_type","resource_id","user_id"],"properties":{"resource_type":{"type":"string","enum":["doc","play","project"]},"resource_id":{"type":"string"},"user_id":{"type":"string"},"actor_id":{"type":"string"}}}`,
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
					"project_id": {"type": "string"},
					"folder_id": {"type": "string", "description": "The project folder the doc lives in."},
					"title": {"type": "string"},
					"body": {"type": "string"},
					"version": {"type": "integer"},
					"archived": {"type": "boolean"},
					"locked": {"type": "boolean"},
					"created_at": {"type": "string", "format": "date-time"},
					"updated_at": {"type": "string", "format": "date-time"}
				}
			},
			"actor_id": {"type": "string"},
			"mentioned_user_ids": {"type": "array", "items": {"type": "string"}, "description": "People this save newly @-mentions, by user id."}
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
					"project_id": {"type": "string"},
					"folder_id": {"type": "string", "description": "The project folder the doc lives in."},
					"title": {"type": "string"},
					"body": {"type": "string"},
					"version": {"type": "integer"},
					"archived": {"type": "boolean"},
					"locked": {"type": "boolean"},
					"created_at": {"type": "string", "format": "date-time"},
					"updated_at": {"type": "string", "format": "date-time"}
				}
			},
			"actor_id": {"type": "string"},
			"mentioned_user_ids": {"type": "array", "items": {"type": "string"}, "description": "People this save newly @-mentions, by user id."},
			"lock_changed": {"type": "boolean", "description": "True when the doc was only locked or unlocked; its title and body are unchanged."}
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
					"assignee": {"type": "string", "deprecated": true, "description": "Deprecated in favour of developer; always carries the same value."},
					"developer": {"type": "string"},
					"tester": {"type": "string"},
					"reporter": {
						"type": "object",
						"required": ["kind"],
						"properties": {
							"kind": {"type": "string", "enum": ["user", "user:mcp", "automation"]},
							"login": {"type": "string"},
							"automation_id": {"type": "string"},
							"automation_name": {"type": "string"}
						}
					},
					"created_at": {"type": "string", "format": "date-time"},
					"updated_at": {"type": "string", "format": "date-time"},
					"finished_at": {"type": ["string", "null"], "format": "date-time"}
				}
			},
			"mentioned_user_ids": {"type": "array", "items": {"type": "string"}, "description": "People the body @-mentions, by user id."}
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
					"trail_id": {"type": "string"},
					"user_id": {"type": "string"}
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
	"notification.push_requested": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["notifications"],
		"properties": {
			"notifications": {
				"type": "array",
				"items": {
					"type": "object",
					"required": ["id", "user_id"],
					"properties": {
						"id": {"type": "string"},
						"user_id": {"type": "string"},
						"workspace_id": {"type": "string"}
					}
				}
			}
		}
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
			"name": {"type": "string"},
			"project_id": {"type": "string"}
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
		"properties": {
			"ticket": {"type": "object"},
			"actor_id": {"type": "string"},
			"mentioned_user_ids": {"type": "array", "items": {"type": "string"}, "description": "People this edit newly @-mentions, by user id."}
		}
	}`,
	"ticket.assignee_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "Deprecated in favour of ticket.developer_changed; still published with the same payload whenever the developer changes.",
		"required": ["ticket", "from", "to"],
		"properties": {
			"ticket": {"type": "object"},
			"from": {"type": "string"},
			"to": {"type": "string"}
		}
	}`,
	"ticket.developer_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["ticket", "from", "to"],
		"properties": {
			"ticket": {"type": "object"},
			"from": {"type": "string"},
			"to": {"type": "string"}
		}
	}`,
	"ticket.tester_changed": `{
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
			"title": {"type": "string"},
			"project_id": {"type": "string"}
		}
	}`,
	"ticket.link_created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "A found-in or blocked-by link was added; ticket_id is found in or blocked by target_id, and an empty target_id on found_in marks the origin unknown.",
		"required": ["link"],
		"properties": {
			"link": {
				"type": "object",
				"required": ["ticket_id", "kind", "target_id"],
				"properties": {
					"ticket_id": {"type": "string"},
					"kind": {"type": "string", "enum": ["found_in", "blocked_by"]},
					"target_id": {"type": "string"},
					"created_at": {"type": "string", "format": "date-time"}
				}
			}
		}
	}`,
	"ticket.link_deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "A found-in or blocked-by link was removed, or replaced by a new found-in.",
		"required": ["link"],
		"properties": {
			"link": {
				"type": "object",
				"required": ["ticket_id", "kind", "target_id"],
				"properties": {
					"ticket_id": {"type": "string"},
					"kind": {"type": "string", "enum": ["found_in", "blocked_by"]},
					"target_id": {"type": "string"},
					"created_at": {"type": "string", "format": "date-time"}
				}
			}
		}
	}`,
	"ticket.test_passed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "A ticket passed testing and moved to a done-stage column; tester is the login of whoever passed it.",
		"required": ["ticket", "tester"],
		"properties": {
			"ticket": {"type": "object"},
			"tester": {"type": "string"}
		}
	}`,
	"ticket.test_failed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "A ticket failed testing and moved back to a progress-stage column; report is the bug report posted to its thread.",
		"required": ["ticket", "tester", "report"],
		"properties": {
			"ticket": {"type": "object"},
			"tester": {"type": "string"},
			"report": {"type": "string"}
		}
	}`,
	"doc.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["id", "title"],
		"properties": {
			"id": {"type": "string"},
			"title": {"type": "string"},
			"project_id": {"type": "string"}
		}
	}`,
	"doc.moved": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "A doc moved to another folder of its project; doc carries its new folder_id.",
		"required": ["doc", "from_folder_id"],
		"properties": {
			"doc": {
				"type": "object",
				"required": ["id", "project_id", "folder_id", "title"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"folder_id": {"type": "string"},
					"title": {"type": "string"}
				}
			},
			"from_folder_id": {"type": "string"},
			"actor_id": {"type": "string"}
		}
	}`,
	"doc.folder.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["folder"],
		"properties": {
			"folder": {
				"type": "object",
				"required": ["id", "project_id", "name", "is_default"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"name": {"type": "string"},
					"is_default": {"type": "boolean", "description": "The project's default folder, where new docs land; it is never deleted."},
					"created_at": {"type": "string", "format": "date-time"},
					"updated_at": {"type": "string", "format": "date-time"}
				}
			},
			"actor_id": {"type": "string"}
		}
	}`,
	"doc.folder.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "A doc folder was renamed.",
		"required": ["folder", "previous_name"],
		"properties": {
			"folder": {
				"type": "object",
				"required": ["id", "project_id", "name", "is_default"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"name": {"type": "string"},
					"is_default": {"type": "boolean", "description": "The project's default folder, where new docs land; it is never deleted."},
					"created_at": {"type": "string", "format": "date-time"},
					"updated_at": {"type": "string", "format": "date-time"}
				}
			},
			"previous_name": {"type": "string"},
			"actor_id": {"type": "string"}
		}
	}`,
	"doc.folder.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "A doc folder was deleted; its docs moved to the project's default folder, never deleted.",
		"required": ["folder", "moved_to_folder_id"],
		"properties": {
			"folder": {
				"type": "object",
				"required": ["id", "project_id", "name", "is_default"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"name": {"type": "string"},
					"is_default": {"type": "boolean", "description": "The project's default folder, where new docs land; it is never deleted."},
					"created_at": {"type": "string", "format": "date-time"},
					"updated_at": {"type": "string", "format": "date-time"}
				}
			},
			"moved_to_folder_id": {"type": "string"},
			"actor_id": {"type": "string"}
		}
	}`,
	"doc.watchers.changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "Someone started or stopped watching a doc, choosing to; a watcher gets the doc's change notifications. Being added for creating or editing the doc rides doc.created and doc.updated instead.",
		"required": ["doc", "user_id", "watching"],
		"properties": {
			"doc": {
				"type": "object",
				"required": ["id", "project_id", "title"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"title": {"type": "string"}
				}
			},
			"user_id": {"type": "string"},
			"watching": {"type": "boolean", "description": "true when they started watching, false when they stopped."}
		}
	}`,
	"doc.clarification.round_started": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "A Clarify via AI run opened a doc's next round of questions; it is being written until the round ends.",
		"required": ["doc", "round", "started_by"],
		"properties": {
			"doc": {
				"type": "object",
				"required": ["id", "project_id", "title"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"title": {"type": "string"}
				}
			},
			"round": {"type": "integer", "minimum": 1},
			"started_by": {"type": "string", "description": "Who started the round's Clarify via AI run."},
			"actor_id": {"type": "string"}
		}
	}`,
	"doc.clarification.round_posted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "A running round posted its questions, or found no gaps left and wrote the doc instead.",
		"required": ["doc", "round", "started_by", "question_count", "no_gaps"],
		"properties": {
			"doc": {
				"type": "object",
				"required": ["id", "project_id", "title"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"title": {"type": "string"}
				}
			},
			"round": {"type": "integer", "minimum": 1},
			"started_by": {"type": "string", "description": "Who started the round's Clarify via AI run."},
			"actor_id": {"type": "string"},
			"question_count": {"type": "integer", "minimum": 0},
			"no_gaps": {"type": "boolean", "description": "true when the round found no gaps left and wrote the doc instead of asking."}
		}
	}`,
	"doc.clarification.round_ended": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "A round's run ended, whatever its outcome; a round that asked nothing and found no gaps is removed.",
		"required": ["doc", "round", "started_by"],
		"properties": {
			"doc": {
				"type": "object",
				"required": ["id", "project_id", "title"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"title": {"type": "string"}
				}
			},
			"round": {"type": "integer", "minimum": 1},
			"started_by": {"type": "string", "description": "Who started the round's Clarify via AI run."},
			"actor_id": {"type": "string"},
			"removed": {"type": "boolean", "description": "true when the round asked nothing and found no gaps, so it is gone."}
		}
	}`,
	"doc.clarification.round_answered": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "The last pending question of a round was answered or skipped; started_by is whom it is for.",
		"required": ["doc", "round", "started_by"],
		"properties": {
			"doc": {
				"type": "object",
				"required": ["id", "project_id", "title"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"title": {"type": "string"}
				}
			},
			"round": {"type": "integer", "minimum": 1},
			"started_by": {"type": "string", "description": "Who started the round's Clarify via AI run."},
			"actor_id": {"type": "string"}
		}
	}`,
	"doc.clarification.answer_saved": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "Someone answered or skipped a question of a doc's clarification; the answer itself never travels.",
		"required": ["doc", "round", "question_id", "question", "author_id"],
		"properties": {
			"doc": {
				"type": "object",
				"required": ["id", "project_id", "title"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"title": {"type": "string"}
				}
			},
			"round": {"type": "integer", "minimum": 1},
			"question_id": {"type": "string"},
			"question": {"type": "string"},
			"author_id": {"type": "string"},
			"at": {"type": "string", "format": "date-time"}
		}
	}`,
	"doc.clarification.answer_cleared": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "Someone made a question of a doc's clarification unanswered again.",
		"required": ["doc", "round", "question_id", "question", "author_id"],
		"properties": {
			"doc": {
				"type": "object",
				"required": ["id", "project_id", "title"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"title": {"type": "string"}
				}
			},
			"round": {"type": "integer", "minimum": 1},
			"question_id": {"type": "string"},
			"question": {"type": "string"},
			"author_id": {"type": "string"},
			"at": {"type": "string", "format": "date-time"}
		}
	}`,
	"doc.clarification.anything_else_saved": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "Someone saved or cleared a round's Anything else? text; the text itself never travels.",
		"required": ["doc", "round", "started_by"],
		"properties": {
			"doc": {
				"type": "object",
				"required": ["id", "project_id", "title"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"title": {"type": "string"}
				}
			},
			"round": {"type": "integer", "minimum": 1},
			"started_by": {"type": "string", "description": "Who started the round's Clarify via AI run."},
			"actor_id": {"type": "string"}
		}
	}`,
	"doc.clarification.closed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"description": "Someone closed a doc's clarification on its newest round; another round reopens it.",
		"required": ["doc", "round", "started_by"],
		"properties": {
			"doc": {
				"type": "object",
				"required": ["id", "project_id", "title"],
				"properties": {
					"id": {"type": "string"},
					"project_id": {"type": "string"},
					"title": {"type": "string"}
				}
			},
			"round": {"type": "integer", "minimum": 1},
			"started_by": {"type": "string", "description": "Who started the round's Clarify via AI run."},
			"actor_id": {"type": "string"}
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
			"label": {"type": "string"},
			"workspace_id": {"type": "string"}
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
			"target_type": {"type": "string", "enum": ["ticket", "doc", "interview"]},
			"target_id": {"type": "string"},
			"target_title": {"type": "string"},
			"starter_id": {"type": "string"},
			"via": {"type": "string", "enum": ["web", "mcp"]},
			"workspace_id": {"type": "string"},
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
			"target_type": {"type": "string", "enum": ["ticket", "doc", "interview"]},
			"target_id": {"type": "string"},
			"target_title": {"type": "string"},
			"starter_id": {"type": "string"},
			"via": {"type": "string", "enum": ["web", "mcp"]},
			"workspace_id": {"type": "string"}
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
			"target_type": {"type": "string", "enum": ["ticket", "doc", "interview"]},
			"target_id": {"type": "string"},
			"target_title": {"type": "string"},
			"starter_id": {"type": "string"},
			"via": {"type": "string", "enum": ["web", "mcp"]},
			"workspace_id": {"type": "string"},
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
					"kind": {"type": "string"},
					"title": {"type": "string"},
					"when_to_use": {"type": "string"},
					"always_included": {"type": "boolean"},
					"footer": {"type": "boolean"},
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
					"kind": {"type": "string"},
					"title": {"type": "string"},
					"when_to_use": {"type": "string"},
					"always_included": {"type": "boolean"},
					"footer": {"type": "boolean"},
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
			"workspace_id": {"type": "string"},
			"project_id": {"type": "string"},
			"title": {"type": "string"},
			"author_id": {"type": "string"}
		}
	}`,
	"interview_template.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["workspace_id", "author_id"],
		"properties": {
			"workspace_id": {"type": "string"},
			"author_id": {"type": "string"},
			"updated_at": {"type": "string", "format": "date-time"}
		}
	}`,
	"interview_answer.saved": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["workspace_id", "project_id", "round", "question", "author_id"],
		"properties": {
			"workspace_id": {"type": "string"},
			"project_id": {"type": "string"},
			"round": {"type": "integer", "minimum": 0},
			"question": {"type": "string"},
			"author_id": {"type": "string"},
			"at": {"type": "string", "format": "date-time"}
		}
	}`,
	"interview_answer.cleared": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["workspace_id", "project_id", "round", "question", "author_id"],
		"properties": {
			"workspace_id": {"type": "string"},
			"project_id": {"type": "string"},
			"round": {"type": "integer", "minimum": 0},
			"question": {"type": "string"},
			"author_id": {"type": "string"},
			"at": {"type": "string", "format": "date-time"}
		}
	}`,
	"interview_source.added": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["workspace_id", "project_id", "source_id", "kind", "stance", "author_id"],
		"properties": {
			"workspace_id": {"type": "string"},
			"project_id": {"type": "string"},
			"source_id": {"type": "string"},
			"kind": {"type": "string", "enum": ["path", "doc", "memory", "project", "text"]},
			"stance": {"type": "string", "enum": ["follow", "question"]},
			"author_id": {"type": "string"},
			"at": {"type": "string", "format": "date-time"}
		}
	}`,
	"interview_source.changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["workspace_id", "project_id", "source_id", "kind", "stance", "author_id"],
		"properties": {
			"workspace_id": {"type": "string"},
			"project_id": {"type": "string"},
			"source_id": {"type": "string"},
			"kind": {"type": "string", "enum": ["path", "doc", "memory", "project", "text"]},
			"stance": {"type": "string", "enum": ["follow", "question"]},
			"author_id": {"type": "string"},
			"at": {"type": "string", "format": "date-time"}
		}
	}`,
	"interview_source.removed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["workspace_id", "project_id", "source_id", "kind", "stance", "author_id"],
		"properties": {
			"workspace_id": {"type": "string"},
			"project_id": {"type": "string"},
			"source_id": {"type": "string"},
			"kind": {"type": "string", "enum": ["path", "doc", "memory", "project", "text"]},
			"stance": {"type": "string", "enum": ["follow", "question"]},
			"author_id": {"type": "string"},
			"at": {"type": "string", "format": "date-time"}
		}
	}`,
	"interview_draft.saved": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["workspace_id", "project_id", "draft_id", "question", "author_id"],
		"properties": {
			"workspace_id": {"type": "string"},
			"project_id": {"type": "string"},
			"draft_id": {"type": "string"},
			"question": {"type": "string"},
			"author_id": {"type": "string"},
			"at": {"type": "string", "format": "date-time"}
		}
	}`,
	"interview_draft.dismissed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["workspace_id", "project_id", "draft_id", "question", "author_id"],
		"properties": {
			"workspace_id": {"type": "string"},
			"project_id": {"type": "string"},
			"draft_id": {"type": "string"},
			"question": {"type": "string"},
			"author_id": {"type": "string"},
			"at": {"type": "string", "format": "date-time"}
		}
	}`,
	"instance_template.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["kind", "key", "author_id", "reset"],
		"properties": {
			"kind": {"type": "string", "enum": ["interview", "mention_chip", "play_instructions", "ticket_body"]},
			"key": {"type": "string"},
			"author_id": {"type": "string"},
			"updated_at": {"type": "string", "format": "date-time"},
			"reset": {"type": "boolean"}
		}
	}`,
	"computer.paired": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["computer_id", "user_id", "server_url", "token_expires_at"],
		"properties": {
			"computer_id": {"type": "string"},
			"user_id": {"type": "string"},
			"server_url": {"type": "string"},
			"harness_version": {"type": "string"},
			"token_expires_at": {"type": "string", "format": "date-time"}
		}
	}`,
	"computer.harness_switched": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["computer_id", "user_id", "from_kind", "to_kind", "harness_version"],
		"properties": {
			"computer_id": {"type": "string"},
			"user_id": {"type": "string"},
			"from_kind": {"type": "string", "description": "The harness kind the computer was stored under, for example t3code."},
			"to_kind": {"type": "string", "description": "The harness kind it moved forward to, for example t3code-v2; a computer never moves back."},
			"harness_version": {"type": "string", "description": "The harness version read when the computer moved."}
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
	"computer.tunnel_created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["computer_id", "user_id", "tunnel_id", "hostname"],
		"properties": {
			"computer_id": {"type": "string"},
			"user_id": {"type": "string"},
			"tunnel_id": {"type": "string"},
			"hostname": {"type": "string"}
		}
	}`,
	"computer.tunnel_removed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["computer_id", "user_id", "tunnel_id", "hostname"],
		"properties": {
			"computer_id": {"type": "string"},
			"user_id": {"type": "string"},
			"tunnel_id": {"type": "string"},
			"hostname": {"type": "string"}
		}
	}`,
	"computer.tunnel_status_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["computer_id", "user_id", "tunnel", "harness_reachable"],
		"properties": {
			"computer_id": {"type": "string"},
			"user_id": {"type": "string"},
			"tunnel": {"type": "string"},
			"harness_reachable": {"type": "boolean"},
			"harness_version": {"type": "string"}
		}
	}`,
	"computer.setup_turn_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["computer_id", "user_id", "run_id", "turn_id", "provider", "provider_name", "state", "status", "started_at"],
		"properties": {
			"computer_id": {"type": "string"},
			"user_id": {"type": "string"},
			"run_id": {"type": "string"},
			"turn_id": {"type": "string"},
			"provider": {"type": "string"},
			"provider_name": {"type": "string"},
			"model": {"type": "string"},
			"state": {"type": "string", "enum": ["running", "confirmed", "failed"]},
			"status": {"type": "string"},
			"started_at": {"type": "string", "format": "date-time"},
			"ended_at": {"type": "string", "format": "date-time"}
		}
	}`,
	"computer.setup_finished": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["computer_id", "user_id", "run_id", "confirmed", "providers"],
		"properties": {
			"computer_id": {"type": "string"},
			"user_id": {"type": "string"},
			"run_id": {"type": "string"},
			"confirmed": {"type": "boolean"},
			"providers": {
				"type": "array",
				"items": {
					"type": "object",
					"required": ["provider", "state", "status"],
					"properties": {
						"provider": {"type": "string"},
						"state": {"type": "string", "enum": ["running", "confirmed", "failed"]},
						"status": {"type": "string"}
					}
				}
			}
		}
	}`,
	"computer.setup_turn_activity": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["computer_id", "user_id", "run_id", "turn_id", "provider", "status"],
		"properties": {
			"computer_id": {"type": "string"},
			"user_id": {"type": "string"},
			"run_id": {"type": "string"},
			"turn_id": {"type": "string"},
			"provider": {"type": "string"},
			"status": {"type": "string"},
			"call_id": {"type": "string"},
			"kind": {"type": "string", "enum": ["tool_call", "tool_result", "text", "question", "other"]},
			"tool": {"type": "string"},
			"text": {"type": "string"},
			"at": {"type": "string"}
		}
	}`,
	"personal_access_token.minted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["token_id", "user_id", "name"],
		"properties": {
			"token_id": {"type": "string"},
			"user_id": {"type": "string"},
			"name": {"type": "string"},
			"computer_id": {"type": "string"}
		}
	}`,
	"personal_access_token.revoked": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["token_id", "user_id", "name"],
		"properties": {
			"token_id": {"type": "string"},
			"user_id": {"type": "string"},
			"name": {"type": "string"},
			"computer_id": {"type": "string"}
		}
	}`,
	"identity.linked": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["user_id", "provider", "login"],
		"properties": {
			"user_id": {"type": "string"},
			"provider": {"type": "string"},
			"login": {"type": "string"}
		}
	}`,
	"identity.unlinked": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["user_id", "provider", "login"],
		"properties": {
			"user_id": {"type": "string"},
			"provider": {"type": "string"},
			"login": {"type": "string"}
		}
	}`,
	"session.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["session_id", "user_id", "client"],
		"properties": {
			"session_id": {"type": "string"},
			"user_id": {"type": "string"},
			"client": {"type": "string", "enum": ["browser", "desktop", "phone"]},
			"platform": {"type": "string"},
			"label": {"type": "string"}
		}
	}`,
	"session.revoked": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["session_id", "user_id", "client"],
		"properties": {
			"session_id": {"type": "string"},
			"user_id": {"type": "string"},
			"client": {"type": "string", "enum": ["browser", "desktop", "phone"]},
			"platform": {"type": "string"},
			"label": {"type": "string"}
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
		"properties": {
			"conversation": {"type": "object"},
			"members_only": {"type": "boolean", "description": "Set on a DM's or a private channel's creation, which is never delivered to integrations or automations."}
		}
	}`,
	"chat.conversation.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["conversation_id", "workspace_id", "kind", "name", "previous_name"],
		"properties": {
			"conversation_id": {"type": "string"},
			"workspace_id": {"type": "string"},
			"kind": {"type": "string", "enum": ["channel", "voice_channel"]},
			"name": {"type": "string"},
			"previous_name": {"type": "string"},
			"actor_id": {"type": "string"},
			"members_only": {"type": "boolean", "description": "Set on a private channel's event, which is never delivered to integrations or automations."}
		}
	}`,
	"chat.conversation.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["conversation_id", "workspace_id", "kind", "name"],
		"properties": {
			"conversation_id": {"type": "string"},
			"workspace_id": {"type": "string"},
			"kind": {"type": "string", "enum": ["channel", "voice_channel"]},
			"name": {"type": "string"},
			"actor_id": {"type": "string"},
			"private": {"type": "boolean"},
			"member_ids": {"type": "array", "items": {"type": "string"}},
			"members_only": {"type": "boolean", "description": "Set on a private channel's event, which is never delivered to integrations or automations."}
		}
	}`,
	"chat.conversation.members_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["conversation_id", "workspace_id", "private", "added_user_ids", "removed_user_ids"],
		"properties": {
			"conversation_id": {"type": "string"},
			"workspace_id": {"type": "string"},
			"private": {"type": "boolean"},
			"added_user_ids": {"type": "array", "items": {"type": "string"}},
			"removed_user_ids": {"type": "array", "items": {"type": "string"}},
			"actor_id": {"type": "string"},
			"members_only": {"type": "boolean", "description": "Set on a private channel's change, a switch to private included, which is never delivered to integrations or automations."}
		}
	}`,
	"chat.message.created": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["message"],
		"properties": {
			"message": {"type": "object", "description": "The message as chat stores it. Its attachment_id is set only on a note: an Agent message on a ticket's thread, posted on the author_id person's behalf, whose markdown file is that attachment of the conversation. Its handoffs are set only on an Agent reply that handed work to other agents: each one's id, driver, model, title, prompt, state (running, done, failed, interrupted or left_running), final reply and steps."},
			"members_only": {"type": "boolean", "description": "Set on a DM or private channel's message, which is never delivered to integrations or automations."}
		}
	}`,
	"chat.message.updated": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["message"],
		"properties": {
			"message": {"type": "object", "description": "The message after the change: an edited body, or a note whose markdown file changed, which moves its updated_at and leaves its body as it was."},
			"members_only": {"type": "boolean", "description": "Set on a DM or private channel's message, which is never delivered to integrations or automations."}
		}
	}`,
	"chat.message.deleted": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["conversation_id", "message_id", "deleted_at"],
		"properties": {
			"conversation_id": {"type": "string"},
			"message_id": {"type": "string"},
			"deleted_at": {"type": "string", "format": "date-time"},
			"members_only": {"type": "boolean", "description": "Set on a DM or private channel's message, which is never delivered to integrations or automations."}
		}
	}`,
	"chat.message.reactions_changed": `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["conversation_id", "message_id", "user_id", "emoji", "reacted"],
		"properties": {
			"conversation_id": {"type": "string"},
			"message_id": {"type": "string"},
			"user_id": {"type": "string", "description": "The person who added or removed the reaction."},
			"emoji": {"type": "string", "description": "The emoji itself, such as 👍."},
			"reacted": {"type": "boolean", "description": "True when the reaction was added, false when it was removed."},
			"members_only": {"type": "boolean", "description": "Set on a DM or private channel's message, which is never delivered to integrations or automations."}
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
			},
			"members_only": {"type": "boolean", "description": "Set on a private voice channel's call, which is never delivered to integrations or automations."}
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
