// An image reference reads by its last path segment: the registry and repository path say little once a stack is open.
// A digest keeps 12 hex characters, the length docker prints.
export const imageShortName = (ref: string): string => {
  const last = ref.slice(ref.lastIndexOf("/") + 1);
  const at = last.indexOf("@sha256:");
  return at < 0 ? last : `${last.slice(0, at)}@${last.slice(at + 8, at + 20)}`;
};

// Where an identifier cut in its middle splits: an image before its tag or digest, the part that changes from deploy
// to deploy; a hostname after the dot that starts its last two labels, so its domain stays whole.
export const keptTail = (value: string, kind: "image" | "hostname"): [string, string] => {
  const at = kind === "image" ? Math.max(value.lastIndexOf(":"), value.lastIndexOf("@")) : value.lastIndexOf(".", value.lastIndexOf(".") - 1) + 1;
  return at <= 0 ? [value, ""] : [value.slice(0, at), value.slice(at)];
};

// "27m" or "2h 5m": an elapsed time short enough to sit after a status word.
export const elapsedShort = (fromIso: string, now: number): string => {
  const minutes = Math.max(0, Math.floor((now - Date.parse(fromIso)) / 60_000));
  if (minutes < 1) return "<1m";
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  return minutes % 60 === 0 ? `${hours}h` : `${hours}h ${minutes % 60}m`;
};
