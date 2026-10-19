import { toast } from "sonner";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api, errorMessage } from "@/api/client";
import { useSessionStore } from "@/stores/sessionStore";
import { useSetupPassStore } from "@/stores/setupPassStore";
import { useViewAsStore } from "@/stores/viewAsStore";

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

describe("errorMessage", () => {
  it.each([
    ["field error wins", { message: "m", code: "c", errors: { title: ["Title is required"] } }, "Title is required"],
    ["falls back to message", { message: "not found", code: "not_found" }, "not found"],
    ["falls back to axios message", { message: "Network Error" }, "Network Error"],
    ["a refusal shows the sentence of the rule behind it", { message: "forbidden: docs:lock required (Denied on doc Spec for role Client)", code: "FORBIDDEN", details: { action: "docs:lock", sentence: "Denied on doc Spec for role Client" } }, "Denied on doc Spec for role Client"],
    ["a refusal with no sentence keeps its message", { message: "forbidden: docs:lock required", code: "FORBIDDEN", details: { sentence: "" } }, "forbidden: docs:lock required"],
    ["falls back to default", undefined, "Something went wrong"],
  ])("%s", (_name, data, expected) => {
    if (data === undefined) {
      expect(errorMessage(undefined)).toBe(expected);
      return;
    }
    const error = { response: { data } };
    expect(errorMessage(error)).toBe(expected);
  });

  it("returns the first field error even when a message exists", () => {
    const error = { response: { data: { message: "validation failed", code: "invalid", errors: { name: ["a"], title: ["b"] } } } };
    expect(errorMessage(error)).toBe("a");
  });
});

const captureAuth = async () => {
  let captured: { headers: Record<string, string> } | undefined;
  await api.get("/ping", {
    adapter: async (config) => {
      captured = config;
      return { data: {}, status: 200, statusText: "OK", headers: {}, config };
    },
  });
  return captured?.headers.Authorization;
};

const FUTURE = "2999-01-01T00:00:00Z";

describe("api client interceptors", () => {
  beforeEach(() => {
    useSessionStore.setState({ token: null, isLoggedIn: false });
    useSetupPassStore.setState({ token: null, expiresAt: null, code: null });
  });

  it("clears a refused setup pass on 401 without redirecting to /login", async () => {
    useSetupPassStore.getState().unlock({ token: "pass-1", expires_at: FUTURE }, "nxs_abc");
    const assignMock = vi.fn();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { pathname: "/", assign: assignMock },
    });
    const error = { response: { status: 401, data: { message: "unauthorized", code: "unauthorized" } }, message: "401" };
    await expect(api.get("/api/dns/zones", { adapter: async () => Promise.reject(error) })).rejects.toMatchObject({
      message: "401",
    });
    expect(useSetupPassStore.getState().token).toBeNull();
    expect(assignMock).not.toHaveBeenCalled();
  });

  it("does not send an expired setup pass", async () => {
    useSetupPassStore.getState().unlock({ token: "pass-1", expires_at: "2000-01-01T00:00:00Z" }, "nxs_abc");
    expect(await captureAuth()).toBeUndefined();
    expect(useSetupPassStore.getState().token).toBeNull();
  });

  it("sends the setup pass as the bearer when there is no session", async () => {
    useSetupPassStore.getState().unlock({ token: "pass-1", expires_at: FUTURE }, "nxs_abc");
    expect(await captureAuth()).toBe("Bearer pass-1");
  });

  it("prefers the session over a leftover setup pass", async () => {
    useSetupPassStore.getState().unlock({ token: "pass-1", expires_at: FUTURE }, "nxs_abc");
    useSessionStore.getState().login("tok-1");
    expect(await captureAuth()).toBe("Bearer tok-1");
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("injects the bearer token from the session store", async () => {
    useSessionStore.getState().login("tok-1");
    let captured: { headers: Record<string, string> } | undefined;
    await api.get("/ping", {
      adapter: async (config) => {
        captured = config;
        return { data: {}, status: 200, statusText: "OK", headers: {}, config };
      },
    });
    expect(captured?.headers.Authorization).toBe("Bearer tok-1");
  });

  it("does not add the header when no token is present", async () => {
    let captured: { headers: Record<string, string> } | undefined;
    await api.get("/ping", {
      adapter: async (config) => {
        captured = config;
        return { data: {}, status: 200, statusText: "OK", headers: {}, config };
      },
    });
    expect(captured?.headers.Authorization).toBeUndefined();
  });

  it("logs out and redirects on a 401 response", async () => {
    useSessionStore.getState().login("tok-1");
    const assignMock = vi.fn();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { pathname: "/", assign: assignMock },
    });
    const error = {
      response: { status: 401, data: { message: "unauthorized", code: "unauthorized" } },
      message: "Request failed",
    };
    await expect(
      api.get("/ping", { adapter: async () => Promise.reject(error) }),
    ).rejects.toMatchObject({ message: "Request failed" });
    expect(useSessionStore.getState().isLoggedIn).toBe(false);
    expect(assignMock).toHaveBeenCalledWith("/login");
  });

  it("does not redirect when already on /login", async () => {
    const assignMock = vi.fn();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { pathname: "/login", assign: assignMock },
    });
    const error = {
      response: { status: 401, data: { message: "unauthorized", code: "unauthorized" } },
      message: "Request failed",
    };
    await expect(
      api.get("/ping", { adapter: async () => Promise.reject(error) }),
    ).rejects.toMatchObject({ message: "Request failed" });
    expect(assignMock).not.toHaveBeenCalled();
  });

  it("does not log out on non-401 errors", async () => {
    useSessionStore.getState().login("tok-1");
    const error = {
      response: { status: 500, data: { message: "boom", code: "internal" } },
      message: "Request failed",
    };
    await expect(
      api.get("/ping", { adapter: async () => Promise.reject(error) }),
    ).rejects.toMatchObject({ message: "Request failed" });
    expect(useSessionStore.getState().isLoggedIn).toBe(true);
  });
});

const alice = { workspaceId: "ws-1", workspaceName: "Acme", userId: "u-alice", name: "Alice", avatarUrl: "" };

const sentHeaders = async () => {
  let captured: Record<string, unknown> = {};
  await api.get("/ping", {
    adapter: async (config) => {
      captured = { ...config.headers };
      return { data: {}, status: 200, statusText: "OK", headers: {}, config };
    },
  });
  return captured;
};

const refuse = (status: number, headers: Record<string, string>) =>
  api.get("/ping", {
    adapter: async () =>
      Promise.reject({ response: { status, headers, data: { message: "Nova now holds permissions you don't", code: "FORBIDDEN" } }, message: "Request failed" }),
  });

describe("api client while viewing as another member", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useSessionStore.getState().login("tok-1");
    useViewAsStore.setState({ target: null });
  });

  it("sends both view headers only while a target is set", async () => {
    expect(await sentHeaders()).not.toHaveProperty("X-View-As-User");

    useViewAsStore.getState().start(alice);
    expect(await sentHeaders()).toMatchObject({ "X-View-As-User": "u-alice", "X-View-As-Workspace": "ws-1" });

    useViewAsStore.getState().exit();
    expect(await sentHeaders()).not.toHaveProperty("X-View-As-Workspace");
  });

  it("leaves the view when the server refuses it, and stays signed in", async () => {
    useViewAsStore.getState().start(alice);
    await expect(refuse(403, { "x-view-as": "refused" })).rejects.toBeDefined();
    expect(useViewAsStore.getState().target).toBeNull();
    expect(useSessionStore.getState().isLoggedIn).toBe(true);
    expect(toast.error).toHaveBeenCalledWith("Stopped viewing as Alice: Nova now holds permissions you don't");
  });

  it("keeps the view on an ordinary 403, such as a write the view turns away", async () => {
    useViewAsStore.getState().start(alice);
    await expect(refuse(403, {})).rejects.toBeDefined();
    expect(useViewAsStore.getState().target).toEqual(alice);
    expect(toast.error).not.toHaveBeenCalled();
  });
});
