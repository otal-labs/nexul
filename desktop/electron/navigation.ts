// Only the launcher, the active instance origin, or github.com (AU1's OAuth dance) may load here.

export const OAUTH_HOSTS = new Set(["github.com"]);

export function isAllowedNavigation(
  targetUrl: string,
  currentUrl: string,
  activeOrigin: string | null,
): boolean {
  let target: URL;
  try {
    target = new URL(targetUrl);
  } catch {
    return false;
  }
  if (target.protocol === "file:") return currentUrl.startsWith("file:");
  if (activeOrigin !== null && target.origin === activeOrigin) return true;
  return OAUTH_HOSTS.has(target.hostname);
}
