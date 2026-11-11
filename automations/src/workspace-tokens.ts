// Prefix of the unit env variable holding the host's token for one workspace's secrets manager; internal/install
// writes the same names (workspaceTokenEnv). "OP" is the manager's reference scheme.
export const TOKEN_ENV_PREFIX = "NEXUL_OP_TOKEN_";

// The variable name carries the workspace id's bytes in upper case hex, so any id maps to one valid name and back.
const ID_HEX = /^(?:[0-9A-F]{2})+$/;

// Removes every workspace token from env and returns them by workspace id, so the worker threads created afterwards,
// which copy the process environment, never inherit one (ADR 0182).
export function takeWorkspaceTokens(env: NodeJS.ProcessEnv = process.env): Map<string, string> {
  const tokens = new Map<string, string>();
  for (const name of Object.keys(env)) {
    if (!name.startsWith(TOKEN_ENV_PREFIX)) continue;
    const value = env[name];
    delete env[name];
    const encoded = name.slice(TOKEN_ENV_PREFIX.length);
    if (value && ID_HEX.test(encoded)) tokens.set(Buffer.from(encoded, "hex").toString("utf8"), value);
  }
  return tokens;
}
