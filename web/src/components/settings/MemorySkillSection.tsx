import { McpSnippetCard } from "@/components/settings/McpSnippetCard";
import { SettingsCard } from "@/components/settings/SettingsCard";

// The memory-protocol skill file: fully static text, so a plain constant instead of a build() function.
const SKILL_FILE = `---
name: nexul-memory
description: Use Nexul's memories — durable notes for agents at workspace or project scope, own entity from docs — before and during any task.
---

# Nexul memory protocol

Memories are durable, human-editable notes for agents, not people: their own
entity with a table, page, and permission, never a doc. Each belongs to the
workspace or to one project. Workspace memories reach every turn; project
memories reach only that project's turns. Every chat turn already carries
the applicable always-included memories inlined in full as standing rules,
plus an index (name + when-to-use only) of the rest; this skill applies the
same protocol in your own local T3 sessions.

## Read

1. Treat any always-included memories already inlined in the turn as
   standing rules — no need to fetch them again.
2. Look at the memories index for the rest (or call \`memory_list\` for the
   current project if none was provided).
3. Pick the memories whose when-to-use line matches what you're about to
   do — don't fetch everything, only what's relevant.
4. Fetch a chosen memory's full content with the \`memory_get\` MCP tool,
   passing its id.

## Write

Save a memory when:

- You learn a durable fact worth remembering for next time (a coding
  convention, a gotcha, a standing preference) — use your own judgment.
- A user says something like "@Agent remember X" — always save when asked,
  even if it seems minor.

Create a new memory with \`memory_create\`: pass a \`project_id\` to scope it to
that project, or omit it to save at workspace scope. Prefer project scope
unless the fact holds for every project. Update an existing one with
\`memory_update\` if one already covers the same ground. Humans can always
edit memories directly on the Memories page — that's the safety valve for a
wrong or stale memory.

## Curate

Keep the set small and specific. Prefer updating an existing memory over
creating a near-duplicate. A when-to-use line should be short enough to
scan in a list: "use this if you are writing React code", not a paragraph.
`;

export const MemorySkillSection = () => (
  <SettingsCard
    id="pairing-memory-skill"
    title="Agent memory skill"
    description="Copy this into your own T3 Code skills to teach any session Nexul's shared memory protocol — the same one Agent follows in chat."
  >
    <McpSnippetCard label="nexul-memory" filename="SKILL.md" snippet={SKILL_FILE} />
  </SettingsCard>
);
