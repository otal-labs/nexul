import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  useArchiveDoc,
  useCreateDoc,
  useFetchDoc,
  useFetchDocs,
  useFetchDocsByProject,
  useRestoreDoc,
} from "@/hooks/DocHooks";

vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const doc = {
  id: "doc-1",
  project_id: "project-1",
  title: "Spec",
  body: "body",
  version: 1,
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
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
  vi.mocked(api.delete).mockReset();
});

describe("useFetchDocs", () => {
  it("loads the doc list", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [doc] });
    const { result } = renderHook(() => useFetchDocs(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([doc]));
    expect(api.get).toHaveBeenCalledWith("/api/docs");
  });
});

describe("useFetchDocsByProject", () => {
  it("loads the project-scoped doc list", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [doc] });
    const { result } = renderHook(() => useFetchDocsByProject("project-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([doc]));
    expect(api.get).toHaveBeenCalledWith("/api/docs", { params: { project_id: "project-1" } });
  });

  it("is disabled without a project id", () => {
    const { result } = renderHook(() => useFetchDocsByProject(""), { wrapper });
    expect(result.current.fetchStatus).toBe("idle");
    expect(api.get).not.toHaveBeenCalled();
  });
});

describe("useFetchDoc", () => {
  it("is disabled without an id", () => {
    const { result } = renderHook(() => useFetchDoc(undefined), { wrapper });
    expect(result.current.isPending).toBe(true);
    expect(api.get).not.toHaveBeenCalled();
  });

  it("loads a single doc", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: doc });
    const { result } = renderHook(() => useFetchDoc("doc-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual(doc));
    expect(api.get).toHaveBeenCalledWith("/api/docs/doc-1");
  });
});

describe("useCreateDoc", () => {
  it("posts and invalidates the list", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: doc });
    const { result } = renderHook(() => useCreateDoc(), { wrapper });
    await result.current.mutateAsync({ project_id: "project-1", title: "Spec", body: "body" });
    expect(api.post).toHaveBeenCalledWith("/api/docs", { project_id: "project-1", title: "Spec", body: "body" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("surfaces an api error", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useCreateDoc(), { wrapper });
    await result.current.mutateAsync({ project_id: "project-1", title: "Spec", body: "body" }).catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useArchiveDoc", () => {
  it("posts the archive and invalidates the single-doc key", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { ...doc, archived: true } });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateSpy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useArchiveDoc(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync("doc-1");
    expect(api.post).toHaveBeenCalledWith("/api/docs/doc-1/archive");
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getDoc", "doc-1"] });
  });
});

describe("useRestoreDoc", () => {
  it("posts the restore and invalidates the single-doc key", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: doc });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateSpy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useRestoreDoc(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync("doc-1");
    expect(api.post).toHaveBeenCalledWith("/api/docs/doc-1/restore");
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getDoc", "doc-1"] });
  });
});

