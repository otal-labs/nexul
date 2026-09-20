import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MentionChip } from "@/components/doc/mention/MentionChip";
import type { MentionChipData } from "@/models/Mention";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get },
  errorMessage: vi.fn(),
}));

const mockTemplate = (mention_chip_template: string) =>
  mocks.get.mockResolvedValue({
    data: { instance_url: "", settings_version: 1, oauth_callback: "", mention_chip_template },
  });

const wrap = (ui: React.ReactNode) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>
  );
};

describe("MentionChip", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    // Default template renders identically to the old hardcoded icon+title+status
    // chip (spec.md section 6, regression safety).
    mockTemplate("{ticket.Ticket} {ticket.Status}");
  });

  it("renders a clickable ticket chip matching today's icon+title+status output for the default template", async () => {
    const chip: MentionChipData = { type: "ticket", id: "t-1", title: "Fix the bug", status: "in_progress", status_label: "In progress", can_open: true };
    render(wrap(<MentionChip type="ticket" id="t-1" label="old" chip={chip} />));

    const link = await screen.findByRole("link", { name: /Fix the bug/ });
    expect(link).toHaveAttribute("href", "/tickets/t-1");
    expect(link).toHaveAttribute("data-can-open", "true");
    expect(link).toHaveTextContent("Fix the bug In progress");
  });

  it("substitutes a custom template's tokens from the resolved chip fields", async () => {
    mockTemplate("{ticket.Project} {ticket.Ticket}");
    const chip: MentionChipData = {
      type: "ticket",
      id: "t-1",
      title: "Fix login redirect loop",
      status_label: "In progress",
      can_open: true,
      project_prefix: "ERF",
      project_number: 1,
    };
    render(wrap(<MentionChip type="ticket" id="t-1" label="old" chip={chip} />));

    const link = await screen.findByRole("link", { name: /ERF-1/ });
    expect(link).toHaveTextContent("ERF-1 Fix login redirect loop");
  });

  it("passes an unrecognized token through literally", async () => {
    mockTemplate("{ticket.Ticket} {ticket.NotAField}");
    const chip: MentionChipData = { type: "ticket", id: "t-1", title: "Fix the bug", can_open: true };
    render(wrap(<MentionChip type="ticket" id="t-1" label="old" chip={chip} />));

    // findByRole matching just the title would also match the pre-fetch fallback
    // render, so wait for the fully substituted text instead (only present once the
    // custom template resolves).
    const text = await screen.findByText("Fix the bug {ticket.NotAField}");
    expect(text.closest("a")).toHaveAttribute("href", "/tickets/t-1");
  });

  it("renders a clickable doc chip linking to the doc, unaffected by the ticket template", async () => {
    mockTemplate("{ticket.Project} {ticket.Ticket}");
    const chip: MentionChipData = { type: "doc", id: "d-9", title: "Architecture", can_open: true };
    render(wrap(<MentionChip type="doc" id="d-9" label="old" chip={chip} />));

    const link = await screen.findByRole("link", { name: "Architecture" });
    expect(link).toHaveAttribute("href", "/docs/d-9");
  });

  it("renders an inert chip with disclosed title when the target cannot be opened", async () => {
    const chip: MentionChipData = { type: "doc", id: "d-2", title: "Secret vault", can_open: false };
    render(wrap(<MentionChip type="doc" id="d-2" label="old" chip={chip} />));

    const inert = await screen.findByTestId("mention-chip");
    expect(inert).toHaveAttribute("data-can-open", "false");
    expect(inert).toHaveTextContent("Secret vault");
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("falls back to the stored label before resolution", async () => {
    render(wrap(<MentionChip type="ticket" id="t-1" label="Stored title" />));
    const inert = await screen.findByTestId("mention-chip");
    expect(inert).toHaveTextContent("Stored title");
    expect(inert).toHaveAttribute("data-can-open", "false");
  });

  it("falls back to the stored label when the target no longer resolves", async () => {
    render(wrap(<MentionChip type="ticket" id="gone" label="Gone ticket" chip={undefined} />));
    const inert = await screen.findByTestId("mention-chip");
    expect(inert).toHaveTextContent("Gone ticket");
  });
});
