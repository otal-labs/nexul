import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AttachmentsSection } from "@/components/attachment/AttachmentsSection";
import { api } from "@/api/client";
import type { Attachment } from "@/models/Attachment";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(() => "error"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const confirmOpen = vi.fn();
vi.mock("@/hooks/useConfirmationDialog", () => ({
  useConfirmationDialog: () => ({ open: confirmOpen }),
}));

const image: Attachment = {
  id: "a-1",
  ticket_id: "t-1",
  name: "shot.png",
  content_type: "image/png",
  size: 2048,
  uploaded_by: "user-1",
  created_at: new Date().toISOString(),
};

const pdf: Attachment = { ...image, id: "a-2", name: "spec.pdf", content_type: "application/pdf", size: 5 * 1024 * 1024 };

const renderSection = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <AttachmentsSection owner={{ ticket_id: "t-1" }} />
    </QueryClientProvider>,
  );

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.delete).mockReset();
  confirmOpen.mockReset();
  globalThis.URL.createObjectURL = vi.fn(() => "blob:shot");
  globalThis.URL.revokeObjectURL = vi.fn();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/attachments") return { data: [image, pdf] };
    return { data: new Blob(["png"]) };
  });
});

describe("AttachmentsSection", () => {
  it("lists files with size, a thumbnail for images, and a count", async () => {
    renderSection();
    expect(await screen.findByText("shot.png")).toBeInTheDocument();
    expect(screen.getByText("spec.pdf")).toBeInTheDocument();
    expect(screen.getByText(/2 KB/)).toBeInTheDocument();
    expect(screen.getByText(/5\.0 MB/)).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Attachments\s*2/ })).toBeInTheDocument();
    await waitFor(() => expect(document.querySelector('img[src="blob:shot"]')).not.toBeNull());
    expect(api.get).toHaveBeenCalledWith("/api/attachments", { params: { ticket_id: "t-1" } });
    // Only the image row fetches bytes for its thumbnail.
    expect(vi.mocked(api.get).mock.calls.filter(([url]) => url === "/api/attachments/a-1")).toHaveLength(1);
    expect(vi.mocked(api.get).mock.calls.filter(([url]) => url === "/api/attachments/a-2")).toHaveLength(0);
  });

  it("shows the empty state when there are no files", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    renderSection();
    expect(await screen.findByText(/No files yet/)).toBeInTheDocument();
  });

  it("uploads a picked file against the owner", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: image });
    renderSection();
    await screen.findByText("shot.png");

    const file = new File(["png"], "new.png", { type: "image/png" });
    await user.upload(screen.getByLabelText("Attachment file"), file);

    await waitFor(() => expect(api.post).toHaveBeenCalledOnce());
    const form = vi.mocked(api.post).mock.calls[0]?.[1] as FormData;
    expect(form.get("ticket_id")).toBe("t-1");
    expect(form.get("file")).toBe(file);
  });

  it("deletes after confirmation and not when cancelled", async () => {
    const user = userEvent.setup();
    vi.mocked(api.delete).mockResolvedValue({});
    renderSection();
    await screen.findByText("shot.png");

    confirmOpen.mockResolvedValueOnce(false);
    await user.click(screen.getByRole("button", { name: "Delete spec.pdf" }));
    expect(api.delete).not.toHaveBeenCalled();

    confirmOpen.mockResolvedValueOnce(true);
    await user.click(screen.getByRole("button", { name: "Delete spec.pdf" }));
    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/attachments/a-2"));
  });

  it("downloads a file through the authenticated client", async () => {
    const user = userEvent.setup();
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    renderSection();
    await screen.findByText("spec.pdf");

    await user.click(screen.getByRole("button", { name: "Download spec.pdf" }));

    await waitFor(() => expect(click).toHaveBeenCalledOnce());
    expect(api.get).toHaveBeenCalledWith("/api/attachments/a-2", { responseType: "blob" });
    click.mockRestore();
  });
});
