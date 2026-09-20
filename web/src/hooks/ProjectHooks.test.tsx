import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  useAddProjectRepo,
  useCreateProject,
  useDeleteProject,
  useFetchProjectDeleteImpact,
  useFetchProjectRepos,
  useFetchProjects,
  useRemoveProjectRepo,
  useRenameProject,
} from "@/hooks/ProjectHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" };

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.delete).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("useFetchProjects", () => {
  it("loads the project list", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [project] });
    const { result } = renderHook(() => useFetchProjects(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([project]));
    expect(api.get).toHaveBeenCalledWith("/api/projects", { params: { workspace_id: "ws-1" } });
  });
});

describe("useFetchProjectDeleteImpact", () => {
  it("loads the delete impact for a project", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { tickets: 2, repos: 1 } });
    const { result } = renderHook(() => useFetchProjectDeleteImpact("p-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual({ tickets: 2, repos: 1 }));
    expect(api.get).toHaveBeenCalledWith("/api/projects/p-1/impact");
  });
});

describe("useFetchProjectRepos", () => {
  it("loads the repos for a project", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [{ owner: "acme", name: "app", full_name: "acme/app" }] });
    const { result } = renderHook(() => useFetchProjectRepos("p-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([{ owner: "acme", name: "app", full_name: "acme/app" }]));
    expect(api.get).toHaveBeenCalledWith("/api/projects/p-1/repos");
  });
});

describe("useCreateProject", () => {
  it("posts and returns the created project", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: project });
    const { result } = renderHook(() => useCreateProject(), { wrapper });
    await result.current.mutateAsync({ name: "Backend", prefix: "BE" });
    expect(api.post).toHaveBeenCalledWith("/api/projects", {
      name: "Backend",
      prefix: "BE",
      workspace_id: "ws-1",
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useRenameProject", () => {
  it("patches the project name", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...project, name: "New" } });
    const { result } = renderHook(() => useRenameProject(), { wrapper });
    await result.current.mutateAsync({ id: "p-1", name: "New" });
    expect(api.patch).toHaveBeenCalledWith("/api/projects/p-1", { name: "New" });
  });

  it("includes icon in the payload when the caller passes one", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...project, icon: "Rocket" } });
    const { result } = renderHook(() => useRenameProject(), { wrapper });
    await result.current.mutateAsync({ id: "p-1", name: "Backend", icon: "Rocket" });
    expect(api.patch).toHaveBeenCalledWith("/api/projects/p-1", { name: "Backend", icon: "Rocket" });
  });
});

describe("useDeleteProject", () => {
  it("deletes the project", async () => {
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
    const { result } = renderHook(() => useDeleteProject(), { wrapper });
    await result.current.mutateAsync("p-1");
    expect(api.delete).toHaveBeenCalledWith("/api/projects/p-1");
  });
});

describe("useAddProjectRepo", () => {
  it("posts the repo association", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const { result } = renderHook(() => useAddProjectRepo(), { wrapper });
    await result.current.mutateAsync({ projectId: "p-1", owner: "acme", name: "app", connectorId: "github" });
    expect(api.post).toHaveBeenCalledWith("/api/projects/p-1/repos", {
      owner: "acme",
      name: "app",
      connector_id: "github",
    });
  });
});

describe("useRemoveProjectRepo", () => {
  it("deletes the repo association", async () => {
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
    const { result } = renderHook(() => useRemoveProjectRepo(), { wrapper });
    await result.current.mutateAsync({ owner: "acme", name: "app" });
    expect(api.delete).toHaveBeenCalledWith("/api/projects/repos/acme/app");
  });
});

