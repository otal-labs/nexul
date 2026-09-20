// Converts between the KEY=value text block the UI edits and the map the backend stores.
export const parseEnv = (raw: string): Record<string, string> => {
  const out: Record<string, string> = {};
  for (const line of raw.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const eq = trimmed.indexOf("=");
    if (eq <= 0) continue;
    out[trimmed.slice(0, eq).trim()] = trimmed.slice(eq + 1).trim();
  }
  return out;
};

export const formatEnv = (env: Record<string, string> | undefined): string =>
  Object.entries(env ?? {})
    .map(([k, v]) => `${k}=${v}`)
    .join("\n");
