package mcp

// instructions is what a client shows its agent at session start, before it loads any tool schema; keep it under
// 2,048 characters, most important first, and never repeat a tool description here.
const instructions = `Nexul is a self-hosted workspace where docs, tickets, chat, and deploys live together. These tools act as the signed-in user, with that user's permissions.

Tool families, named object_verb:
- Planning: project_* (project_get lists a project's statuses, categories, ticket types, labels, and repositories), ticket_*, doc_*, memory_*.
- Conversation: conversation_list, message_list, message_post, mention_search, notification_*.
- Shipping: stack_* (a stack is what gets deployed), deploy_*, machine_*, gateway_*, exposure_*, dns_*, topology_*, repository_*, pull_request_*.
- Agents and automation: play_*, trail_* (a trail is the record of one play run), automation_*, computer_* (paired computers and their setup).
- Administration: account_*, invitation_*, access_grant_*, instance_*, dead_letter_*.

Workflows:
- Filing work: call project_get for valid status, ticket type, and category ids, then ticket_create. Change a ticket with ticket_update.
- Every update is a patch: send only the fields you mean to change; omitted fields keep their values.
- Shipping: stack_deploy starts a deploy and returns its id. Poll deploy_get (status and log tail) and stack_get (service health) until it settles.

Ids and lists: ids are opaque strings; ticket tools also accept a ticket's key, such as REF-102. Lists return items, total, has_more, and next_offset; pass next_offset back as offset for the next page.

Memories: before a task, check memory_list for notes whose when-to-use matches and read them with memory_get. Save durable facts with memory_create, or memory_update when one already covers the ground.

Deploys and other work started through these tools are recorded as started over MCP by you.`
