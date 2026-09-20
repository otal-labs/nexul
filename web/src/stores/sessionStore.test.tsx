import { beforeEach, describe, expect, it } from "vitest";

import { useSessionStore } from "@/stores/sessionStore";

describe("sessionStore", () => {
  beforeEach(() => {
    useSessionStore.setState({ token: null, isLoggedIn: false });
    useSessionStore.persist.clearStorage();
    localStorage.clear();
  });

  it("starts logged out", () => {
    expect(useSessionStore.getState().isLoggedIn).toBe(false);
    expect(useSessionStore.getState().token).toBeNull();
  });

  it("login stores the token and marks the session active", () => {
    useSessionStore.getState().login("tok-1");
    expect(useSessionStore.getState().isLoggedIn).toBe(true);
    expect(useSessionStore.getState().token).toBe("tok-1");
  });

  it("logout clears the token and session", () => {
    useSessionStore.getState().login("tok-1");
    useSessionStore.getState().logout();
    expect(useSessionStore.getState().isLoggedIn).toBe(false);
    expect(useSessionStore.getState().token).toBeNull();
  });

  it("partializes only data fields into storage", () => {
    useSessionStore.getState().login("tok-1");
    const stored = JSON.parse(localStorage.getItem("session") ?? "{}");
    expect(stored.state.token).toBe("tok-1");
    expect(stored.state.isLoggedIn).toBe(true);
  });
});
