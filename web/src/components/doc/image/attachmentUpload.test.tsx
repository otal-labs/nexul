import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocBodyView } from "@/components/doc/DocBodyView";
import { RichTextEditor } from "@/components/doc/RichTextEditor";
import { bodyToMarkdown, emptyDocJson, parseBodyToJSON } from "@/utils/RichtextUtility";
import type { AttachmentOwner } from "@/models/Attachment";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(() => "error"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const withImage = `{"type":"doc","content":[{"type":"image","attrs":{"src":"/api/attachments/a-1","alt":"shot.png"}}]}`;

const renderEditor = (onChange: (json: string) => void, attachTo?: AttachmentOwner) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <RichTextEditor value={emptyDocJson} onChange={onChange} {...(attachTo ? { attachTo } : {})} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

const pasteFile = (file: File) => {
  const body = screen.getByLabelText(/doc body/i);
  fireEvent.paste(body, {
    clipboardData: { files: [file], items: [], types: ["Files"], getData: () => "" },
  });
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.post).mockResolvedValue({ data: { chips: [] } });
  globalThis.URL.createObjectURL = vi.fn(() => "blob:shot");
});

describe("attachment upload in the editor", () => {
  it("uploads a pasted image against the owner and inserts the stored image node", async () => {
    const onChange = vi.fn();
    vi.mocked(api.post).mockImplementation(async (url: string) => {
      if (url === "/api/attachments") {
        return { data: { id: "a-9", ticket_id: "t-1", name: "image.png", content_type: "image/png", size: 3 } };
      }
      return { data: { chips: [] } };
    });
    vi.mocked(api.get).mockResolvedValue({ data: new Blob(["png"]) });
    renderEditor(onChange, { ticket_id: "t-1" });

    pasteFile(new File(["png"], "image.png", { type: "image/png" }));

    await waitFor(() => {
      const emitted = onChange.mock.calls.at(-1)?.[0] as string;
      expect(emitted).toContain('"src":"/api/attachments/a-9"');
    });
    const form = vi.mocked(api.post).mock.calls.find(([url]) => url === "/api/attachments")?.[1] as FormData;
    expect(form.get("ticket_id")).toBe("t-1");
    expect(form.get("name")).toBe("image.png");
    await waitFor(() => expect(document.querySelector('img[src="blob:shot"]')).not.toBeNull());
  });

  it("inserts a link instead of an image for non-image files", async () => {
    const onChange = vi.fn();
    vi.mocked(api.post).mockImplementation(async (url: string) => {
      if (url === "/api/attachments") {
        return { data: { id: "a-2", doc_id: "d-1", name: "spec.pdf", content_type: "application/pdf", size: 3 } };
      }
      return { data: { chips: [] } };
    });
    renderEditor(onChange, { doc_id: "d-1" });

    pasteFile(new File(["pdf"], "spec.pdf", { type: "application/pdf" }));

    await waitFor(() => {
      const emitted = onChange.mock.calls.at(-1)?.[0] as string;
      expect(emitted).toContain('"href":"/api/attachments/a-2"');
      expect(emitted).toContain("spec.pdf");
    });
  });

  it("ignores pasted files when the editor has no owner", async () => {
    const onChange = vi.fn();
    renderEditor(onChange);

    pasteFile(new File(["png"], "image.png", { type: "image/png" }));

    await new Promise((r) => setTimeout(r, 20));
    expect(vi.mocked(api.post).mock.calls.filter(([url]) => url === "/api/attachments")).toHaveLength(0);
  });

  it("renders stored images in the read-only body through the authenticated client", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: new Blob(["png"]) });
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <DocBodyView body={withImage} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const img = await screen.findByAltText("shot.png");
    expect(img).toHaveAttribute("src", "blob:shot");
    expect(api.get).toHaveBeenCalledWith("/api/attachments/a-1", { responseType: "blob" });
  });

  it("shows a placeholder when the bytes cannot be loaded", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("gone"));
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <DocBodyView body={withImage} />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(await screen.findByRole("img", { name: "shot.png" })).toHaveTextContent("Image unavailable");
  });

  it("round-trips an image through markdown", () => {
    expect(bodyToMarkdown(withImage)).toBe("![shot.png](/api/attachments/a-1)");
    expect(parseBodyToJSON("![shot.png](/api/attachments/a-1)")).toMatchObject(JSON.parse(withImage));
  });
});
