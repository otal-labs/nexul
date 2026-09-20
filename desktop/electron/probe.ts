// Any HTTP response means the server's there; an expired token short-circuits without a network call.

export type ProbeOutcome =
  | { readonly kind: "connected" }
  | { readonly kind: "unreachable"; readonly error: string }
  | { readonly kind: "expired" };

export type FetchLike = (url: string, init: RequestInit) => Promise<Response>;

export async function probeInstance(
  origin: string,
  nowSec: number,
  expiresAtSec: number,
  fetchImpl: FetchLike = fetch,
  timeoutMs = 5000,
): Promise<ProbeOutcome> {
  if (nowSec >= expiresAtSec) return { kind: "expired" };
  try {
    await fetchImpl(`${origin}/api/auth/me`, {
      method: "GET",
      signal: AbortSignal.timeout(timeoutMs),
      redirect: "follow",
    });
    return { kind: "connected" };
  } catch (err) {
    // jsdom's DOMException isn't always `instanceof Error`, so check both.
    const name = err instanceof Error || err instanceof DOMException ? err.name : undefined;
    const error =
      name === "TimeoutError" || name === "AbortError"
        ? `no response within ${timeoutMs}ms`
        : err instanceof Error
          ? err.message
          : "network error";
    return { kind: "unreachable", error };
  }
}
