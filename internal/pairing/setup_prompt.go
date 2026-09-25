package pairing

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/skills"
)

// setupPrompt is what one provider's setup sessions need to know.
type setupPrompt struct {
	ComputerID   string
	ComputerName string
	Driver       string
	ProviderName string
	MCPURL       string
	Token        string
}

// skillLocations are where the default set goes: Claude Code and opencode read the first, Codex and opencode the second.
const skillLocations = "~/.claude/skills/ and ~/.agents/skills/"

// prepareInstructions is the first session: connect Nexul's MCP server and install the skills, both idempotently.
func prepareInstructions(p setupPrompt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are running Nexul's setup for %s on this computer (%s). Work unattended: never ask a question; if a step fails, stop and say which step failed and why.\n\n", p.ProviderName, p.ComputerName)
	fmt.Fprintf(&b, "1. Connect Nexul's MCP server to %s at user level, named `nexul`, replacing any existing `nexul` entry so the token below is the one it uses. It is a Streamable HTTP server at %s, authenticated with the header `Authorization: Bearer %s`.\n", p.ProviderName, p.MCPURL, p.Token)
	b.WriteString(mcpConfigStep(p))
	fmt.Fprintf(&b, "\n2. Install the default skill set, mattpocock/skills, into both %s Never overwrite a skill directory that already exists in either location: the user may have edited it.\n", skillLocations)
	b.WriteString("   - In a new empty temporary directory, run `npx -y skills@latest add mattpocock/skills --skill '*' --agent claude-code --copy --yes`. It writes the skills under `.claude/skills/` in that directory.\n")
	b.WriteString("   - Copy each skill directory it wrote into ~/.claude/skills/ and into ~/.agents/skills/, creating them if needed and skipping any name already present, then delete the temporary directory.\n")
	memory := skills.NexulMemory
	fmt.Fprintf(&b, "3. Install the nexul-memory skill at %s and at %s: write the file at the end of these instructions, exactly as given, wherever it is missing or its frontmatter's `metadata.version` is not %q. An older copy is Nexul's own, so replace it.\n", memory.Paths()[0], memory.Paths()[1], memory.Version)
	fmt.Fprintf(&b, "4. Verify that every skill from step 2 and nexul-memory has a SKILL.md in both %s\n\n", skillLocations)
	b.WriteString("End with one line saying what you connected, installed, and verified.\n\n")
	b.WriteString("The nexul-memory SKILL.md:\n\n````markdown\n")
	b.WriteString(memory.Content)
	b.WriteString("````\n")
	return b.String()
}

// mcpConfigStep says where this provider keeps its MCP servers; an unknown driver is pointed at its own user-level config.
func mcpConfigStep(p setupPrompt) string {
	driver := strings.ToLower(p.Driver)
	if strings.HasPrefix(driver, "claude") {
		return fmt.Sprintf("   Use the CLI: `claude mcp remove --scope user nexul` (ignore a not-found error), then `claude mcp add --scope user --transport http nexul %s --header \"Authorization: Bearer %s\"`.\n", p.MCPURL, p.Token)
	}
	if driver == "codex" {
		return fmt.Sprintf("   Edit ~/.codex/config.toml so it has exactly one `[mcp_servers.nexul]` table:\n\n   ```toml\n   [mcp_servers.nexul]\n   url = \"%s\"\n   http_headers = { Authorization = \"Bearer %s\" }\n   ```\n", p.MCPURL, p.Token)
	}
	if driver == "opencode" {
		return fmt.Sprintf("   Edit ~/.config/opencode/opencode.json, keeping everything else in it, so its `mcp` object has this `nexul` entry:\n\n   ```json\n   \"nexul\": { \"type\": \"remote\", \"url\": \"%s\", \"headers\": { \"Authorization\": \"Bearer %s\" }, \"enabled\": true }\n   ```\n", p.MCPURL, p.Token)
	}
	if strings.HasPrefix(driver, "cursor") {
		return fmt.Sprintf("   Edit ~/.cursor/mcp.json, keeping everything else in it, so its `mcpServers` object has this `nexul` entry:\n\n   ```json\n   \"nexul\": { \"url\": \"%s\", \"headers\": { \"Authorization\": \"Bearer %s\" } }\n   ```\n", p.MCPURL, p.Token)
	}
	return fmt.Sprintf("   Add it to %s's own user-level MCP configuration, through its CLI if it has an MCP command, else its config file, keeping every other entry.\n", p.ProviderName)
}

// confirmInstructions is the second, fresh session: it loads the new MCP entry and skills, so what it reports is what the provider sees.
func confirmInstructions(p setupPrompt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are finishing Nexul's setup for %s on this computer (%s). Work unattended: never ask a question.\n\n", p.ProviderName, p.ComputerName)
	fmt.Fprintf(&b, "1. Check that %s each hold the mattpocock/skills set and nexul-memory, every skill with a SKILL.md.\n", skillLocations)
	b.WriteString("2. List the skills this session has discovered: the ones your harness made available to you, not the files on disk.\n")
	fmt.Fprintf(&b, "3. Call Nexul's MCP tool `computer_setup_update` with computer_id %q, provider %q, confirmed true, and skills set to that list. Do not confirm if Nexul's MCP tools are not available in this session or nexul-memory is missing from the list; say what is missing instead.\n", p.ComputerID, p.Driver)
	fmt.Fprintf(&b, "4. Once the provider is confirmed, confirm the computer itself: call `computer_setup_update` again with computer_id %q and confirmed true, and no provider; confirming again is harmless.\n", p.ComputerID)
	b.WriteString("\nEnd with one line saying what you confirmed.\n")
	return b.String()
}

// mcpURLFor derives the MCP endpoint from the instance URL, the same origin at /mcp; "" if the URL is unusable.
func mcpURLFor(instanceURL string) string {
	u, err := url.Parse(strings.TrimSpace(instanceURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/mcp"
}
