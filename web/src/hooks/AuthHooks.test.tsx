import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  useCompleteFirstLogin,
  useCompleteOwnerWizard,
  useFetchMe,
  useFetchSettings,
  useCopyConnectionToken,
  useUpdateSettings,
  authFollower,
} from "@/hooks/AuthHooks";
import type { MeResponse } from "@/models/User";
import { useDeviceArrivalStore } from "@/stores/deviceArrivalStore";
import { followFrame, isStale, seeded } from "@/test/followFrame";

vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const user = {
  id: "u1",
  provider: "github" as const,
  provider_user_id: "42",
  login: "onik97",
  name: "Onik",
  avatar_url: "https://avatar/x",
  first_login_done: false,
  created_at: "2026-08-12T12:00:00Z",
};

const me: MeResponse = {
  user,
  needs_owner_wizard: false,
  needs_first_login_wizard: true,
  instance_permissions: [],
};

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.put).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.delete).mockReset();
});

describe("useFetchMe", () => {
  it("loads the current user + onboarding status", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: me });
    const { result } = renderHook(() => useFetchMe(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual(me));
    expect(api.get).toHaveBeenCalledWith("/api/auth/me");
  });
});

describe("useCompleteOwnerWizard", () => {
  it("posts the instance URL and invalidates /me", async () => {
    const completed = { ...me, needs_owner_wizard: false, needs_first_login_wizard: false };
    vi.mocked(api.post).mockResolvedValue({ data: completed });
    const { result } = renderHook(() => useCompleteOwnerWizard(), { wrapper });
    await result.current.mutateAsync("https://deploy.example.com");
    expect(api.post).toHaveBeenCalledWith("/api/auth/onboarding/owner", {
      instance_url: "https://deploy.example.com",
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("surfaces an api error", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useCompleteOwnerWizard(), { wrapper });
    await result.current.mutateAsync("https://deploy.example.com").catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useCompleteFirstLogin", () => {
  it("posts the profile completion", async () => {
    const completed = { ...me, needs_first_login_wizard: false };
    vi.mocked(api.post).mockResolvedValue({ data: completed });
    const { result } = renderHook(() => useCompleteFirstLogin(), { wrapper });
    await result.current.mutateAsync();
    expect(api.post).toHaveBeenCalledWith("/api/auth/onboarding/profile");
  });
});

describe("useFetchSettings", () => {
  it("loads instance settings", async () => {
    const settings = {
      instance_url: "https://deploy.example.com",
      settings_version: 2,
      oauth_callback: "https://deploy.example.com/auth/callback",
    };
    vi.mocked(api.get).mockResolvedValue({ data: settings });
    const { result } = renderHook(() => useFetchSettings(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual(settings));
    expect(api.get).toHaveBeenCalledWith("/api/auth/settings");
  });
});

describe("useUpdateSettings", () => {
  it("puts the new instance URL", async () => {
    vi.mocked(api.put).mockResolvedValue({ data: { instance_url: "https://new.example.com", settings_version: 3 } });
    const { result } = renderHook(() => useUpdateSettings(), { wrapper });
    await result.current.mutateAsync("https://new.example.com");
    expect(api.put).toHaveBeenCalledWith("/api/auth/settings", { instance_url: "https://new.example.com" });
  });
});

describe("useCopyConnectionToken", () => {
  it("mints a token and puts it on the clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true, writable: true });
    vi.mocked(api.post).mockResolvedValue({ data: { token: "t", instance_url: "https://deploy.example.com", settings_version: 2, expires_at: "2026-09-11T12:00:00Z" } });
    const { result } = renderHook(() => useCopyConnectionToken(), { wrapper });
    await result.current.mutateAsync();
    expect(api.post).toHaveBeenCalledWith("/api/auth/connection-token");
    expect(writeText).toHaveBeenCalledWith("t");
  });
});

describe("the auth follower", () => {
  const signedIn = () =>
    seeded([
      [["getMe"], { user: { id: "u-1" }, instance_permissions: [] }],
      [["getSessions"], { sessions: [] }],
    ]);

  it("records only the viewer's own phone as it connects, and refetches the devices list for every session", async () => {
    useDeviceArrivalStore.setState({ arrivals: [] });
    const client = signedIn();
    const connect = (payload: Record<string, string>) => followFrame(authFollower, "session.created", payload, client);
    await connect({ session_id: "s-browser", user_id: "u-1", client: "browser", platform: "Linux", label: "Chrome" });
    await connect({ session_id: "s-other", user_id: "u-2", client: "phone", platform: "Android", label: "Pixel 8" });
    await connect({ session_id: "s-phone", user_id: "u-1", client: "phone", platform: "Android", label: "Pixel 8" });
    expect(useDeviceArrivalStore.getState().arrivals.map((a) => a.id)).toEqual(["s-phone"]);
    expect(isStale(client, ["getSessions"])).toBe(true);
  });

  it("refetches the viewer's own account only when it is their name or picture that changed", async () => {
    const client = signedIn();
    await followFrame(authFollower, "account.profile_updated", { account_id: "u-2" }, client);
    expect(isStale(client, ["getMe"])).toBe(false);
    await followFrame(authFollower, "account.profile_updated", { account_id: "u-1" }, client);
    expect(isStale(client, ["getMe"])).toBe(true);
  });
});
