import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  downloadAttachment,
  uploadAttachment,
  useAttachmentBlob,
  useDeleteAttachment,
  useFetchAttachments,
  useUploadAttachment,
} from "@/hooks/AttachmentHooks";
import type { Attachment } from "@/models/Attachment";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(() => "error"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const attachment: Attachment = {
  id: "a-1",
  ticket_id: "t-1",
  name: "shot.png",
  content_type: "image/png",
  size: 1234,
  uploaded_by: "user-1",
  created_at: "2026-08-28T12:00:00Z",
};

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.delete).mockReset();
  globalThis.URL.createObjectURL = vi.fn(() => "blob:shot");
  globalThis.URL.revokeObjectURL = vi.fn();
});

describe("useFetchAttachments", () => {
  it("lists by owner", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [attachment] });
    const { result } = renderHook(() => useFetchAttachments({ ticket_id: "t-1" }), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([attachment]));
    expect(api.get).toHaveBeenCalledWith("/api/attachments", { params: { ticket_id: "t-1" } });
  });
});

describe("uploadAttachment", () => {
  it("posts multipart with the owner, file, and optional name", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: attachment });
    const file = new File(["png"], "shot.png", { type: "image/png" });

    const got = await uploadAttachment({ doc_id: "d-1" }, file, "Pasted.png");

    expect(got).toEqual(attachment);
    const [url, form] = vi.mocked(api.post).mock.calls[0] as [string, FormData];
    expect(url).toBe("/api/attachments");
    expect(form.get("doc_id")).toBe("d-1");
    expect(form.get("file")).toBe(file);
    expect(form.get("name")).toBe("Pasted.png");
  });
});

describe("useUploadAttachment / useDeleteAttachment", () => {
  it("uploads via the mutation", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: attachment });
    const { result } = renderHook(() => useUploadAttachment(), { wrapper });
    const file = new File(["png"], "shot.png", { type: "image/png" });
    await act(async () => {
      await result.current.mutateAsync({ owner: { ticket_id: "t-1" }, file });
    });
    const form = vi.mocked(api.post).mock.calls[0]?.[1] as FormData;
    expect(form.get("ticket_id")).toBe("t-1");
    expect(form.get("name")).toBeNull();
  });

  it("deletes by id", async () => {
    vi.mocked(api.delete).mockResolvedValue({});
    const { result } = renderHook(() => useDeleteAttachment(), { wrapper });
    await act(async () => {
      await result.current.mutateAsync("a-1");
    });
    expect(api.delete).toHaveBeenCalledWith("/api/attachments/a-1");
  });
});

describe("useAttachmentBlob", () => {
  it("fetches the bytes as a blob and returns an object URL", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: new Blob(["png"]) });
    const { result } = renderHook(() => useAttachmentBlob("/api/attachments/a-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toBe("blob:shot"));
    expect(api.get).toHaveBeenCalledWith("/api/attachments/a-1", { responseType: "blob" });
  });

  it("stays idle without a source", () => {
    const { result } = renderHook(() => useAttachmentBlob(null), { wrapper });
    expect(result.current.fetchStatus).toBe("idle");
    expect(api.get).not.toHaveBeenCalled();
  });
});

describe("downloadAttachment", () => {
  it("clicks a temporary anchor named after the file and revokes the URL", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: new Blob(["png"]) });
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

    await downloadAttachment(attachment);

    expect(click).toHaveBeenCalledOnce();
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:shot");
    click.mockRestore();
  });
});
