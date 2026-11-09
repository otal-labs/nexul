import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";

import { DocBodyView } from "@/components/doc/DocBodyView";
import { RichTextEditor } from "@/components/doc/RichTextEditor";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn().mockResolvedValue({ data: { chips: [] } }), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

const page = "```html-page\n<h1>Release</h1>\n```";

const wrap = (node: React.ReactNode) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>{node}</MemoryRouter>
    </QueryClientProvider>,
  );
};

const post = (frame: HTMLIFrameElement, data: unknown, source: MessageEventSource | null = frame.contentWindow) =>
  act(() => {
    window.dispatchEvent(new MessageEvent("message", { data, source }));
  });

afterEach(() => vi.restoreAllMocks());

describe("HtmlPageFrame", () => {
  it("runs a read-only doc's page on an opaque origin and hides its source", async () => {
    wrap(<DocBodyView body={page} />);

    const frame = await screen.findByTitle<HTMLIFrameElement>("HTML page");
    expect(frame.getAttribute("sandbox")).toBe("allow-scripts");
    expect(frame.getAttribute("srcdoc")).toMatch(/<style id="nexul-html-page">.*--foreground:.*<\/style>.*<h1>Release<\/h1>/s);
    expect(document.querySelector(".code-block-view__well")).toHaveClass("hidden");
  });

  it("folds a written page's source while editing until it is asked for", async () => {
    wrap(<RichTextEditor value={page} onChange={() => {}} />);
    await screen.findByTitle("HTML page");
    expect(document.querySelector(".code-block-view pre")).toHaveClass("hidden");

    await userEvent.click(screen.getByRole("button", { name: "Show code" }));

    expect(document.querySelector(".code-block-view pre")).not.toHaveClass("hidden");
    expect(screen.getByTitle("HTML page")).toBeInTheDocument();
  });

  it("opens a new, empty page on its source so it can be written", () => {
    wrap(<RichTextEditor value={"```html-page\n```"} onChange={() => {}} />);

    expect(document.querySelector(".code-block-view pre")).not.toHaveClass("hidden");
    expect(screen.queryByRole("button", { name: "Show code" })).toBeNull();
  });

  it("draws no frame for an empty page", () => {
    wrap(<DocBodyView body={"```html-page\n```"} />);

    expect(screen.queryByTitle("HTML page")).toBeNull();
  });

  it("fits its height to what its own page reports, never another window's", async () => {
    wrap(<DocBodyView body={page} />);
    const frame = await screen.findByTitle<HTMLIFrameElement>("HTML page");

    post(frame, { source: "nexul-html-page", type: "height", height: 320 });
    post(frame, { source: "nexul-html-page", type: "height", height: 900 }, window);

    expect(frame.style.height).toBe("320px");
  });

  it("opens a link only while the reader is using the frame", async () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    wrap(<DocBodyView body={page} />);
    const frame = await screen.findByTitle<HTMLIFrameElement>("HTML page");
    const link = { source: "nexul-html-page", type: "link", url: "https://example.com/notes" };

    post(frame, link);
    expect(open).not.toHaveBeenCalled();

    vi.spyOn(document, "activeElement", "get").mockReturnValue(frame);
    post(frame, link);
    expect(open).toHaveBeenCalledWith("https://example.com/notes", "_blank", "noopener,noreferrer");
  });
});
