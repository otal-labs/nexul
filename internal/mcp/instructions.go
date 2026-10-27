package mcp

// instructions is what a client shows its agent at session start, before it loads any tool schema; keep it under
// 2,048 characters, most important first, and never repeat a tool description here.
const instructions = `Nexul is a self-hosted workspace for docs, tickets, chat, and deploys. Tools act as the signed-in user, with their permissions. A browser does not share that sign-in: unless one is signed in separately, read records here and use supplied screenshots as visual evidence; a logged-out page's not found means nothing. A person may see only some of a workspace's projects, so not found may mean no access.

Tool families, named object_verb:
- Planning: workspace_list (the workspace ids other tools need) and workspace_update, project_* (project_get lists its statuses, categories, ticket types, labels, repositories, doc folders), ticket_*, doc_*, memory_*.
- Conversation: conversation_*, message_list, message_post, botwebhook_* (bots), mention_search, notification_*.
- Shipping: stack_* (a stack is what gets deployed), deploy_*, machine_*, gateway_*, exposure_*, dns_*, topology_*, repository_*, pull_request_*.
- Agents and automation: play_*, trail_* (a trail records one play run), automation_* (per workspace), computer_* (paired computers and their setup).
- Administration: account_*, invitation_*, role_* (workspace_list with an id lists them), permission_overwrite_*, instance_*, template_* (defaults), dead_letter_*.

Workflows:
- Every update is a patch: omitted fields keep their value.
- Files: attachment_create returns a markdown line for a body or message; attachment_get reads one.
- Shipping: stack_deploy returns a deploy id; poll deploy_list with that id and stack_get until it settles.

Ids are opaque; ticket tools also accept a key such as REF-102, plus workspace (id or slug) when two of yours share it. Lists return items, total, has_more, and next_offset (pass it back as offset); a list given an id returns that one record. A field's description lists its valid values.

Memories: before a task, check memory_list for notes whose when-to-use fits, read them with memory_get; save durable facts with memory_create, or memory_update the one that covers them.

Work started here is recorded as started via MCP.`
