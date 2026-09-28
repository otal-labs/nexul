import { useEffect } from "react";

import { registerPushToken } from "@/push/pushToken";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

// Fires after sign-in (and on cold start with a persisted session); registerPushToken no-ops without a project id.
export const usePushRegistration = (): void => {
  const signedIn = useSessionStore((s) => s.signedIn);
  const host = useSessionStore((s) => s.host);

  useEffect(() => {
    if (!signedIn || !host) return;
    const token = readSessionToken();
    if (!token) return;
    void registerPushToken(host, token);
  }, [signedIn, host]);
};
