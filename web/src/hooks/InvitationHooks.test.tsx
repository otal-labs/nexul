import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  useCreateInvitation,
  useCreateInvitationAcceptance,
  useFetchInvitationPreview,
  useRedeemInvitation,
  useStartInvitationOAuth,
} from "@/hooks/InvitationHooks";

vi.mock("@/api/client", () => ({ api: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() }, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{children}</QueryClientProvider>;

describe("InvitationHooks", () => {
  it("previews with a POST body so the token never becomes a path or query", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { instance_name: "Nexul", grants: [], expires_at: "later", providers: ["github"] } });
    const { result } = renderHook(() => useFetchInvitationPreview("raw-token"), { wrapper });
    await waitFor(() => expect(result.current.data?.instance_name).toBe("Nexul"));
    expect(api.post).toHaveBeenCalledWith("/api/invitations/preview", { token: "raw-token" });
  });

  it("starts OAuth through the JSON handoff and returns the provider URL", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { url: "https://oauth.example/authorize" } });
    const { result } = renderHook(() => useStartInvitationOAuth(), { wrapper });
    await result.current.mutateAsync({ provider: "google", token: "raw-token" });
    expect(api.post).toHaveBeenCalledWith("/api/invitations/oauth", { provider: "google", token: "raw-token" });
  });

  it("creates and redeems through typed JSON bodies", async () => {
    vi.mocked(api.post)
      .mockResolvedValueOnce({ data: { id: "i-1", url: "https://nexul/invite#token", grants: [], expires_at: "later" } })
      .mockResolvedValueOnce({ data: { token: "session", workspace_ids: ["ws-1"] } });
    const { result } = renderHook(() => ({ create: useCreateInvitation(), redeem: useRedeemInvitation() }), { wrapper });
    await result.current.create.mutateAsync({ expires_in_days: 7, grants: [{ workspace_id: "ws-1", role_id: "r-1", allow: [], deny: [] }] });
    await result.current.redeem.mutateAsync("acceptance-token");
    expect(api.post).toHaveBeenNthCalledWith(1, "/api/invitations", expect.objectContaining({ expires_in_days: 7 }));
    expect(api.post).toHaveBeenNthCalledWith(2, "/api/invitations/redeem", { acceptance_token: "acceptance-token" });
  });

  it("exchanges the acceptance fragment credential for full details", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { instance_name: "Nexul", grants: [], expires_at: "later", providers: [], acceptance_token: "acceptance-token" } });
    const { result } = renderHook(() => useCreateInvitationAcceptance(), { wrapper });
    await result.current.mutateAsync({ acceptance_token: "acceptance-token" });
    expect(api.post).toHaveBeenCalledWith("/api/invitations/acceptance", { acceptance_token: "acceptance-token" });
  });
});
