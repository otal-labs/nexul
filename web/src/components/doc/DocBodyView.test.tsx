import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocBodyView } from "@/components/doc/DocBodyView";
import { publishChips } from "@/components/doc/mention/mentionChipsStore";
import type { MentionChipData } from "@/models/Mention";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

const structuredWithMention = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"see "},{"type":"mention","attrs":{"type":"ticket","id":"t-1","label":"old title"}},{"type":"text","text":" now"}]}]}`;

const chips: MentionChipData[] = [
  { type: "ticket", id: "t-1", title: "Fix the bug", status: "in_progress", status_label: "In progress", can_open: true },
  { type: "doc", id: "d-2", title: "Secret vault", can_open: false },
];

const renderBody = (body: string) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DocBodyView body={body} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  // MentionChip reads the workspace mention-chip-layout template via useFetchSettings
  // (spec.md section 6, ticket 09); default template renders identically to the old
  // hardcoded title+status chip.
  vi.mocked(api.get).mockResolvedValue({
    data: {
      instance_url: "",
      settings_version: 1,
      oauth_callback: "",
      mention_chip_template: "{ticket.Ticket} {ticket.Status}",
    },
  });
  vi.mocked(api.post).mockReset();
  publishChips(new Map());
});

describe("DocBodyView", () => {
  it("renders prose from structured JSON", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { chips } });
    renderBody(structuredWithMention);
    await screen.findByRole("link", { name: /Fix the bug/ });
    expect(document.querySelector(".tiptap")?.textContent).toContain("see");
  });

  it("renders prose from legacy markdown", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { chips } });
    renderBody("# Heading\n\nSome **bold** body");
    expect(await screen.findByRole("heading", { name: "Heading" })).toBeInTheDocument();
    expect(document.querySelector(".tiptap")?.textContent).toContain("Some bold body");
  });

  it("resolves all mention refs of the render in one batched request", async () => {
    const body = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"mention","attrs":{"type":"ticket","id":"t-1","label":"a"}},{"type":"text","text":" "},{"type":"mention","attrs":{"type":"doc","id":"d-2","label":"b"}}]}]}`;
    vi.mocked(api.post).mockResolvedValue({ data: { chips } });
    renderBody(body);

    await waitFor(() => {
      expect(api.post).toHaveBeenCalledWith("/api/mentions/resolve", {
        refs: [
          { type: "ticket", id: "t-1" },
          { type: "doc", id: "d-2" },
        ],
      });
    });
    expect(api.post).toHaveBeenCalledTimes(1);
  });

  it("hydrates chips with live title, status label, and clickability", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { chips } });
    renderBody(structuredWithMention);

    const link = await screen.findByRole("link", { name: /Fix the bug/ });
    expect(link).toHaveAttribute("href", "/tickets/t-1");
    expect(link).toHaveTextContent("Fix the bug In progress");
  });

  it("keeps inaccessible chips inert with title disclosed", async () => {
    const body = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"mention","attrs":{"type":"doc","id":"d-2","label":"old"}}]}]}`;
    vi.mocked(api.post).mockResolvedValue({ data: { chips } });
    renderBody(body);

    const inert = await screen.findByTestId("mention-chip");
    expect(inert).toHaveAttribute("data-can-open", "false");
    expect(inert).toHaveTextContent("Secret vault");
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("parses canonical markdown internal links in legacy bodies into live chips", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { chips } });
    renderBody("per [Fix the bug](/tickets/t-1) now");

    const link = await screen.findByRole("link", { name: /Fix the bug/ });
    expect(link).toHaveAttribute("href", "/tickets/t-1");
  });

  it("leaves external links as plain links", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { chips } });
    renderBody("see [site](https://example.com) please");

    await screen.findByRole("link", { name: "site" });
    expect(screen.getByRole("link", { name: "site" })).toHaveAttribute("href", "https://example.com");
    expect(api.post).not.toHaveBeenCalled();
  });

  it("resolves nothing for bodies without mentions", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { chips } });
    renderBody("just prose");

    await screen.findByText("just prose");
    expect(api.post).not.toHaveBeenCalled();
  });
});
