import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api, errorMessage } from "@/api/client";
import { useSessionStore } from "@/stores/sessionStore";
import { useSetupPassStore } from "@/stores/setupPassStore";

describe("errorMessage", () => {
  it.each([
    ["field error wins", { message: "m", code: "c", errors: { title: ["Title is required"] } }, "Title is required"],
    ["falls back to message", { message: "not found", code: "not_found" }, "not found"],
    ["falls back to axios message", { message: "Network Error" }, "Network Error"],
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
