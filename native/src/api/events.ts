// The phone reaches its instance by the host it signed in to, with the token from the secure store.
export const buildLiveURL = (host: string, token: string): string =>
  `${host.replace(/^http/i, "ws")}/ws/events?token=${encodeURIComponent(token)}`;
