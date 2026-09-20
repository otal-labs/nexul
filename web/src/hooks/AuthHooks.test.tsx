import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  useAddMember,
  useCompleteFirstLogin,
  useCompleteOwnerWizard,
  useFetchMe,
  useFetchMembers,
  useFetchSettings,
  useGenerateConnectionToken,
  useLookupMembers,
  useRemoveMember,
  useUpdateMentionChipTemplate,
  useUpdateSettings,
} from "@/hooks/AuthHooks";
import type { MeResponse } from "@/models/User";

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
  can_create_workspace: false,
  first_login_done: false,
  created_at: "2026-08-12T12:00:00Z",
};

const me: MeResponse = {
  user,
  needs_owner_wizard: false,
  needs_first_login_wizard: true,
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
      mention_chip_template: "{ticket.Ticket} {ticket.Status}",
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

describe("useUpdateMentionChipTemplate", () => {
  it("patches the mention chip template", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { mention_chip_template: "{ticket.Project} {ticket.Ticket}" } });
    const { result } = renderHook(() => useUpdateMentionChipTemplate(), { wrapper });
    await result.current.mutateAsync("{ticket.Project} {ticket.Ticket}");
    expect(api.patch).toHaveBeenCalledWith("/api/auth/settings/mention-chip-template", {
      mention_chip_template: "{ticket.Project} {ticket.Ticket}",
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("surfaces an api error", async () => {
    vi.mocked(api.patch).mockRejectedValue(new Error("forbidden"));
    const { result } = renderHook(() => useUpdateMentionChipTemplate(), { wrapper });
    await result.current.mutateAsync("{ticket.Ticket}").catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useGenerateConnectionToken", () => {
  it("posts to generate a token", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { token: "t", instance_url: "https://deploy.example.com", settings_version: 2, expires_at: "2026-09-11T12:00:00Z" } });
    const { result } = renderHook(() => useGenerateConnectionToken(), { wrapper });
    await result.current.mutateAsync();
    expect(api.post).toHaveBeenCalledWith("/api/auth/connection-token");
  });
});

describe("useFetchMembers", () => {
  it("loads the allowlist", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { members: ["alice", "bob"] } });
    const { result } = renderHook(() => useFetchMembers(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual({ members: ["alice", "bob"] }));
    expect(api.get).toHaveBeenCalledWith("/api/auth/members");
  });
});

describe("useAddMember", () => {
  it("posts a member and returns the list", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { members: ["bob"] } });
    const { result } = renderHook(() => useAddMember(), { wrapper });
    await result.current.mutateAsync("bob");
    expect(api.post).toHaveBeenCalledWith("/api/auth/members", { login: "bob" });
  });
});

describe("useRemoveMember", () => {
  it("deletes a member by login", async () => {
    vi.mocked(api.delete).mockResolvedValue({ data: { members: [] } });
    const { result } = renderHook(() => useRemoveMember(), { wrapper });
    await result.current.mutateAsync("bob");
    expect(api.delete).toHaveBeenCalledWith("/api/auth/members/bob");
  });
});

describe("useLookupMembers", () => {
  it("looks up GitHub username matches", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { matches: [{ login: "octocat", avatar_url: "https://avatar/octocat" }] },
    });
    const { result } = renderHook(() => useLookupMembers("oct"), { wrapper });
    await waitFor(() =>
      expect(result.current.data).toEqual({ matches: [{ login: "octocat", avatar_url: "https://avatar/octocat" }] }),
    );
    expect(api.get).toHaveBeenCalledWith("/api/auth/members/lookup", expect.objectContaining({ params: { q: "oct" } }));
  });

  it("only fetches the last query when the text changes inside the debounce window", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { matches: [] } });
    const { result, rerender } = renderHook(({ q }) => useLookupMembers(q), { wrapper, initialProps: { q: "oc" } });
    rerender({ q: "oct" });
    await waitFor(() => expect(result.current.data).toEqual({ matches: [] }));
    expect(api.get).toHaveBeenCalledTimes(1);
    expect(api.get).toHaveBeenCalledWith("/api/auth/members/lookup", expect.objectContaining({ params: { q: "oct" } }));
  });

  it("is disabled for a query shorter than 2 characters", () => {
    const { result } = renderHook(() => useLookupMembers("o"), { wrapper });
    expect(result.current.isFetching).toBe(false);
    expect(api.get).not.toHaveBeenCalled();
  });

  it("is disabled for an email-shaped query", () => {
    const { result } = renderHook(() => useLookupMembers("client@example.com"), { wrapper });
    expect(result.current.isFetching).toBe(false);
    expect(api.get).not.toHaveBeenCalled();
  });
});
