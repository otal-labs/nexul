package mcp

// instructions is what a client shows its agent at session start, before it loads any tool schema; keep it under
// 2,048 characters, most important first, and never repeat a tool description here.
const instructions = `Nexul is a self-hosted workspace where docs, tickets, chat, and deploys live together. Tools act as the signed-in user, with their permissions. A browser does not share that sign-in: unless one is signed in separately, read records here and use supplied screenshots as visual evidence; a logged-out page's not found proves nothing. A person may see only some of a workspace's projects, so not found can mean no access, not a fault.

Tool families, named object_verb:
- Planning: workspace_list (the workspace ids other tools need) and workspace_update, project_* (project_get lists a project's statuses, categories, ticket types, labels, repositories, doc folders), ticket_*, doc_*, memory_*.
- Conversation: conversation_*, message_list, message_post, botwebhook_* (bots), mention_search, notification_*.
- Shipping: stack_* (a stack is what gets deployed), deploy_*, machine_*, gateway_*, exposure_*, dns_*, topology_*, repository_*, pull_request_*.
- Agents and automation: play_*, trail_* (a trail is the record of one play run), automation_* (per workspace), computer_* (paired computers, their setup).
- Administration: account_*, invitation_*, role_* (workspace_list with an id lists them), permission_overwrite_*, instance_*, template_* (defaults), dead_letter_*.

Workflows:
- Every update is a patch: omitted fields keep their values.
- Files: attachment_create returns markdown for a line of a body or message; attachment_get reads one.
- Shipping: stack_deploy returns a deploy id; poll deploy_get and stack_get until it settles.

Ids are opaque strings; ticket tools also accept a key such as REF-102, plus workspace (id or slug) when two of yours share it. Lists return items, total, has_more, and next_offset (pass it back as offset).

Memories: before a task, check memory_list for notes whose when-to-use fits; read them with memory_get. Save durable facts with memory_create, or memory_update the one that covers them.

Work you start here is recorded as started via MCP.`
