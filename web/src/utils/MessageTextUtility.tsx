const toURL = (raw: string): URL | null => {
  try {
    return new URL(raw);
  } catch {
    return null;
  }
};

export interface InstanceLink {
  // Path, query, and hash, for in-app navigation.
  path: string;
  workspace: string | undefined;
  section: string | undefined;
  rest: string[];
}

const unscopedRoots = new Set(["settings", "login", "invite", "setup", "wizard"]);

// A URL on this instance's own origin, split into its workspace slug and the page under it; null for anywhere else.
export const parseInstanceLink = (url: string, origin: string): InstanceLink | null => {
  const parsed = toURL(url);
  if (!parsed || parsed.origin !== origin) return null;
  const path = `${parsed.pathname}${parsed.search}${parsed.hash}`;
  const segments = parsed.pathname.split("/").filter((s) => s !== "");
  const [first, second, ...rest] = segments;
  if (first === undefined || unscopedRoots.has(first)) return { path, workspace: undefined, section: first, rest: segments.slice(1) };
  return { path, workspace: first, section: second, rest };
};
