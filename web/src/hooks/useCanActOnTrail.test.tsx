import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useCanActOnTrail } from "@/hooks/useCanActOnTrail";
import type { Trail } from "@/models/Trail";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asReader } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "error" }));

const trail = { starter_id: "u-alice", project_id: "p-1" } as Trail;

const wrapper = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{children}</QueryClientProvider>
);

const answered = (viewer: string, permissions: string[]) => {
  mocks.get.mockImplementation(asReader({ permissions }, (url) => (url === "/api/auth/me" ? { user: { id: viewer } } : [])));
  return renderHook(() => useCanActOnTrail(trail), { wrapper });
};

beforeEach(() => {
  mocks.get.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("useCanActOnTrail", () => {
  it.each([
    { name: "the starter", viewer: "u-alice", permissions: [], expected: true },
    { name: "a plays:write holder", viewer: "u-bob", permissions: ["plays:write"], expected: true },
    { name: "anyone else", viewer: "u-bob", permissions: ["plays:read"], expected: false },
  ])("answers $expected for $name", async ({ viewer, permissions, expected }) => {
    const { result } = answered(viewer, permissions);
    await waitFor(() => expect(result.current).toBe(expected));
  });
});
