package mcp

// instructions is what a client shows its agent at session start, before it loads any tool schema; keep it under
// 2,048 characters, most important first, and never repeat a tool description here.
const instructions = `Nexul is a self-hosted workspace where docs, tickets, chat, and deploys live together. These tools act as the signed-in user, with that user's permissions.

Tool families, named object_verb:
- Planning: workspace_list (the workspace ids other tools need) and workspace_update, project_* (project_get lists a project's statuses, categories, ticket types, labels, repositories, doc folders), ticket_*, doc_* (each doc is in one folder, Main by default), memory_*.
- Conversation: conversation_* (conversation_update renames a channel), message_list, message_post, mention_search, notification_*.
- Shipping: stack_* (a stack is what gets deployed), deploy_*, machine_*, gateway_*, exposure_*, dns_*, topology_*, repository_*, pull_request_*.
- Agents and automation: play_*, trail_* (a trail is the record of one play run), automation_* (each automation belongs to one workspace; the decisions check is a per-workspace switch, off by default, set by play_update), computer_* (paired computers, their setup).
- Administration: account_*, invitation_*, role_* (workspace_list with an id lists roles and valid permissions), permission_overwrite_*, instance_*, dead_letter_*.

Workflows:
- Filing work: call project_get for valid status, ticket type, and category ids, then ticket_create. Change a ticket with ticket_update.
- Every update is a patch: omitted fields keep their values.
- Shipping: stack_deploy returns a deploy id; poll deploy_get (status, log tail) and stack_get (service health) until it settles.

Ids and lists: ids are opaque strings; ticket tools also accept a ticket key such as REF-102, unique within a workspace: pass workspace (id or slug) when two of yours share it. Lists return items, total, has_more, and next_offset, which you pass back as offset.

Memories: before a task, check memory_list for notes whose when-to-use fits; read them with memory_get. Save durable facts with memory_create, or memory_update the one that covers them.

Work you start through these tools, deploys included, is recorded as started via MCP.`
