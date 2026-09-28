import type { ConnectLink } from "@/models/Connect";

const decode = (value: string): string | null => {
  try {
    return decodeURIComponent(value.replace(/\+/g, " "));
  } catch {
    return null;
  }
};

// The React Native URL polyfill has no working searchParams, so the query is split by hand.
const parseQuery = (query: string): Record<string, string> =>
  Object.fromEntries(
    query
      .split("&")
      .filter(Boolean)
      .map((pair) => {
        const at = pair.indexOf("=");
        const key = at === -1 ? pair : pair.slice(0, at);
        const value = at === -1 ? "" : pair.slice(at + 1);
        return [decode(key) ?? key, decode(value) ?? ""];
      }),
  );

// A typed address defaults to https; a trailing slash is dropped so paths append cleanly.
export const normalizeHost = (raw: string): string => {
  const trimmed = raw.trim().replace(/\/+$/, "");
  if (trimmed === "") return "";
  if (/^https?:\/\//i.test(trimmed)) return trimmed;
  return `https://${trimmed}`;
};

export const parseConnectLink = (raw: string): ConnectLink | null => {
  const match = /^nexul:\/\/connect\/?\?(.*)$/i.exec(raw.trim());
  if (!match) return null;
  const params = parseQuery(match[1] ?? "");
  const host = normalizeHost(params.host ?? "");
  const code = (params.code ?? "").trim();
  if (host === "" || code === "") return null;
  return { host, code };
};
