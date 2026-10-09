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
  docFollower,
} from "@/hooks/DocHooks";
import type { DocListItem } from "@/models/Doc";
import { followFrame, isStale, seeded } from "@/test/followFrame";

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

describe("the doc follower", () => {
  const item = (id: string, folder: string, project = "p-1") => ({ ...doc, id, folder_id: folder, project_id: project, can_open: true });
  const views = () =>
    seeded([
      [["getDocs"], [item("d-1", "f-1"), item("d-2", "f-1")]],
      [["getDocs", "byProject", "p-1"], [item("d-1", "f-1"), item("d-2", "f-1")]],
      [["getDocs", "byProject", "p-2"], [item("d-3", "f-9", "p-2")]],
      [["getDoc", "d-1"], item("d-1", "f-1")],
      [["getDocClarification", "d-1"], { rounds: [] }],
      [["getDocClarification", "d-2"], { rounds: [] }],
    ]);
  const folders = (client: ReturnType<typeof views>, key: unknown[]) => client.getQueryData<DocListItem[]>(key)?.map((d) => d.folder_id);

  it("moves a doc to its new folder in every view from the frame alone", async () => {
    const client = views();
    await followFrame(docFollower, "doc.moved", { doc: { ...doc, id: "d-1", project_id: "p-1", folder_id: "f-2" }, from_folder_id: "f-1" }, client);
    expect(folders(client, ["getDocs", "byProject", "p-1"])).toEqual(["f-2", "f-1"]);
    expect(client.getQueryData<DocListItem>(["getDoc", "d-1"])?.folder_id).toBe("f-2");
  });

  it("moves a deleted folder's docs into the folder the frame names", async () => {
    const client = views();
    await followFrame(docFollower, "doc.folder.deleted", { folder: { id: "f-1", project_id: "p-1" }, moved_to_folder_id: "f-0" }, client);
    expect([folders(client, ["getDocs"]), folders(client, ["getDocs", "byProject", "p-2"])]).toEqual([["f-0", "f-0"], ["f-9"]]);
    expect(client.getQueryData<DocListItem>(["getDoc", "d-1"])?.folder_id).toBe("f-0");
  });

  it("refetches the lists a new doc belongs to and one doc's clarification, and leaves the rest", async () => {
    const client = views();
    await followFrame(docFollower, "doc.created", { doc: { ...doc, id: "d-4", project_id: "p-1" } }, client);
    await followFrame(docFollower, "doc.clarification.round_posted", { doc: { id: "d-2" }, round: 1 }, client);
    const keys = [["getDocs"], ["getDocs", "byProject", "p-1"], ["getDocs", "byProject", "p-2"], ["getDocClarification", "d-1"], ["getDocClarification", "d-2"]];
    expect(keys.map((key) => isStale(client, key))).toEqual([true, true, false, false, true]);
  });
});
