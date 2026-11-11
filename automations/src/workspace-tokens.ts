// Prefix of the unit env variable holding the host's token for one workspace's secrets manager; internal/install
// writes the same names (workspaceTokenEnv). "OP" is the manager's reference scheme.
export const TOKEN_ENV_PREFIX = "NEXUL_OP_TOKEN_";

// A workspace slug is lower case words joined by dashes; the variable name upper-cases it and swaps dashes for underscores.
const SLUG_PART = /^[A-Z0-9]+(_[A-Z0-9]+)*$/;

// Removes every workspace token from env and returns them by slug, so the worker threads created afterwards, which copy
// the process environment, never inherit one (ADR 0182).
export function takeWorkspaceTokens(env: NodeJS.ProcessEnv = process.env): Map<string, string> {
  const tokens = new Map<string, string>();
  for (const name of Object.keys(env)) {
    if (!name.startsWith(TOKEN_ENV_PREFIX)) continue;
    const value = env[name];
    delete env[name];
    const part = name.slice(TOKEN_ENV_PREFIX.length);
    if (value && SLUG_PART.test(part)) tokens.set(part.toLowerCase().replaceAll("_", "-"), value);
  }
  return tokens;
}
