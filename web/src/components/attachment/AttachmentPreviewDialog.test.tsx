import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AttachmentPreviewDialog } from "@/components/attachment/AttachmentPreviewDialog";
import { api } from "@/api/client";
import type { Attachment } from "@/models/Attachment";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: vi.fn(() => "error") }));

const file = (name: string, content_type: string): Attachment => ({
  id: "a-1",
  ticket_id: "t-1",
  name,
  content_type,
  size: 2048,
  uploaded_by: "user-1",
  created_at: new Date().toISOString(),
});

const blobs: Blob[] = [];

const openPreview = async (attachment: Attachment) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <AttachmentPreviewDialog attachment={attachment}>
        <button type="button">{attachment.name}</button>
      </AttachmentPreviewDialog>
    </QueryClientProvider>,
  );
  expect(api.get).not.toHaveBeenCalled();
  await userEvent.setup().click(screen.getByRole("button", { name: attachment.name }));
  return screen.findByRole("dialog", { name: attachment.name });
};

beforeEach(() => {
  blobs.length = 0;
  vi.mocked(api.get).mockReset();
  vi.mocked(api.get).mockResolvedValue({ data: new Blob(["bytes"], { type: "text/plain" }) });
  globalThis.URL.createObjectURL = vi.fn((blob: Blob) => {
    blobs.push(blob);
    return "blob:preview";
  });
  globalThis.URL.revokeObjectURL = vi.fn();
});

describe("AttachmentPreviewDialog", () => {
  it("shows a PDF in a frame, its bytes labelled as a PDF", async () => {
    await openPreview(file("spec.pdf", "application/pdf"));
    expect(await screen.findByTitle("spec.pdf")).toHaveAttribute("src", "blob:preview");
    expect(blobs.map((b) => b.type)).toEqual(["application/pdf"]);
  });

  it("shows an SVG the server serves as text through an image, labelled as SVG", async () => {
    await openPreview(file("diagram.svg", "text/xml; charset=utf-8"));
    expect(await screen.findByRole("img", { name: "diagram.svg" })).toHaveAttribute("src", "blob:preview");
    expect(blobs.map((b) => b.type)).toEqual(["image/svg+xml"]);
  });

  it("plays audio in place", async () => {
    await openPreview(file("voice.mp3", "audio/mpeg"));
    expect(await screen.findByLabelText("Play voice.mp3")).toHaveAttribute("src", "blob:preview");
  });

  it("shows a CSV's header and first rows, and says when it was cut", async () => {
    const lines = ["name,role", ...Array.from({ length: 150 }, (_, i) => `person${i},dev`)];
    vi.mocked(api.get).mockResolvedValue({ data: lines.join("\n") });
    await openPreview(file("people.csv", "text/plain; charset=utf-8"));
    expect(await screen.findByRole("columnheader", { name: "name" })).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "person0" })).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "person99" })).toBeInTheDocument();
    expect(screen.queryByRole("cell", { name: "person100" })).not.toBeInTheDocument();
    expect(screen.getByText(/Showing up to 100 rows/)).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledWith("/api/attachments/a-1", { responseType: "text" });
  });

  it("offers the download for a file it can't show, without fetching it", async () => {
    const dialog = await openPreview(file("dump.sqlite", "application/octet-stream"));
    expect(dialog).toHaveTextContent("No preview for this type of file");
    expect(api.get).not.toHaveBeenCalled();
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    await userEvent.setup().click(screen.getByRole("button", { name: "Download" }));
    expect(api.get).toHaveBeenCalledWith("/api/attachments/a-1", { responseType: "blob" });
    expect(click).toHaveBeenCalled();
    click.mockRestore();
  });
});
