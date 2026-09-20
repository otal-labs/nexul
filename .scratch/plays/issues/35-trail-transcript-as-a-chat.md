# 35 — The trail transcript reads as a chat

**What to build:** The trail dialog keeps its facts header, but the transcript below it is a conversation, not a list of tool rows: the starter's "Started <play>" message with their instructions, then the Agent's turn as it happened, prose segments the Agent wrote between actions, each tool call as a collapsible row inline with that prose, the question card where the Agent asked something and the starter's answer as their own message, and the final reply as prose. System notes (a skipped move, a stop) are muted lines between turns. The pipeline records the Agent's interleaved text as steps so the transcript can show what it said before and after each action, and the harness's built-in tools (commands, file changes) carry their command or path in the row label.

**Blocked by:** None — can start immediately

**Status:** done

- [ ] The transcript shows the starter's message first, the Agent's prose between tool calls in order, each tool call as a collapsible row inside the Agent's turn, and the final reply as prose, for a fresh run
- [ ] A run that asked a question shows the question card in place and the starter's answer as their message
- [ ] Command and file-change rows name the command or the path
- [ ] The thread's running line and the trail row summaries still work; existing trails render (legacy string steps as plain rows)

## Design lock

The look is the AI-chat thread the owner pointed at: user messages as compact bubbles on the right; the assistant's turn on the left with no bubble, prose in the body text style with markdown; inside the turn, each action is one collapsible row on its own line, a small kind icon, the label in the body size (tool name plus a short argument preview, or the command, or the path), a state suffix after a middle dot when it failed or was denied, a chevron; expanded, the row shows the arguments and the result as mono blocks; a "Reasoning" row is the collapsed narration when a text segment is only a sentence of intent before an action. Rows use the animated running state and done or failed marks already built for the transcript. Prose segments and rows are separated by the same tight rhythm as chat messages. Mono Console tokens; color only for state.
