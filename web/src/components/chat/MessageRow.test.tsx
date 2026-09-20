import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { MessageRow } from "@/components/chat/MessageRow";
import type { AuthorKind, Message } from "@/models/Chat";
import type { Trail } from "@/models/Trail";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn() },
  errorMessage: vi.fn(() => "error"),
}));

const message = (overrides: Partial<Message>): Message => ({
  id: "m1",
  conversation_id: "c1",
  author_id: "u1",
  author_kind: "user" as AuthorKind,
  body: "hello",
  mentions: null,
  created_at: "2026-08-26T00:00:00Z",
  updated_at: "2026-08-26T00:00:00Z",
  ...overrides,
});

const noop = async () => {};

const renderRow = (ui: ReactNode) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
});

describe("MessageRow author_kind rendering", () => {
  it("renders your own message as a right-aligned bubble, no name header, with edit/delete controls", () => {
    renderRow(<MessageRow message={message({})} authorLogin="onik97" isOwn onEdit={noop} onDelete={noop} />);
    expect(screen.queryByText("onik97")).not.toBeInTheDocument();
    expect(document.querySelector('[data-slot="message"]')?.getAttribute("data-align")).toBe("end");
    expect(screen.getByLabelText("Edit message")).toBeInTheDocument();
    expect(screen.getByLabelText("Delete message")).toBeInTheDocument();
  });

  it("renders a teammate's message left-aligned with their login header", () => {
    renderRow(<MessageRow message={message({ author_id: "u2" })} authorLogin="lena" isOwn={false} onEdit={noop} onDelete={noop} />);
    expect(screen.getByText("lena")).toBeInTheDocument();
    expect(document.querySelector('[data-slot="message"]')?.getAttribute("data-align")).toBe("start");
    expect(screen.queryByLabelText("Edit message")).not.toBeInTheDocument();
  });

  it("renders an agent message as 'Agent' with an App badge and 'via <login>' attribution", () => {
    renderRow(
      <MessageRow
        message={message({ author_kind: "agent", author_id: "u1", body: "Here's the deploy status." })}
        authorLogin="onik97"
        isOwn
        onEdit={noop}
        onDelete={noop}
      />,
    );
    expect(screen.getByText("Agent")).toBeInTheDocument();
    expect(screen.getByText("App")).toBeInTheDocument();
    expect(screen.getByText("via onik97")).toBeInTheDocument();
    expect(screen.getByText("Here's the deploy status.")).toBeInTheDocument();
  });

  it("never shows edit/delete on an agent message, even when author_id matches the viewer", () => {
    renderRow(
      <MessageRow message={message({ author_kind: "agent", author_id: "u1" })} authorLogin="onik97" isOwn onEdit={noop} onDelete={noop} />,
    );
    expect(screen.queryByLabelText("Edit message")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Delete message")).not.toBeInTheDocument();
  });

  it("renders a system note as a muted line with no avatar and no author header", () => {
    const onEdit = vi.fn();
    renderRow(
      <MessageRow
        message={message({ author_kind: "system", body: "@Agent needs a paired T3 Code computer." })}
        authorLogin="onik97"
        isOwn={false}
        onEdit={onEdit}
        onDelete={noop}
      />,
    );
    expect(screen.getByText("@Agent needs a paired T3 Code computer.")).toBeInTheDocument();
    expect(screen.queryByText("onik97")).not.toBeInTheDocument();
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Edit message")).not.toBeInTheDocument();
  });
});

describe("MessageRow attachment image rendering", () => {
  it("renders an attachment image line as an <img> with the alt text and no raw markdown text", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: new Blob(["png"]) });
    globalThis.URL.createObjectURL = vi.fn(() => "blob:shot");
    renderRow(
      <MessageRow
        message={message({ body: "look at this\n![shot.png](/api/attachments/a-9)" })}
        authorLogin="onik97"
        isOwn={false}
        onEdit={noop}
        onDelete={noop}
      />,
    );
    const img = await screen.findByAltText("shot.png");
    expect(img).toHaveAttribute("src", "blob:shot");
    expect(screen.getByText("look at this")).toBeInTheDocument();
    expect(screen.queryByText(/!\[shot\.png\]/)).not.toBeInTheDocument();
  });
});

describe("MessageRow as the Agent's turn", () => {
  it("renders an agent reply as markdown prose, no bubble", () => {
    renderRow(
      <MessageRow message={message({ author_kind: "agent", body: "Ran `go test`, all green." })} authorLogin="onik97" isOwn onEdit={noop} onDelete={noop} />,
    );
    expect(screen.getByText("go test", { selector: "code" })).toBeInTheDocument();
    expect(document.querySelector('[data-slot="bubble"]')).toBeNull();
  });

  it("renders the run's turns above the reply when a trail led to it", async () => {
    const trail = { id: "tr-1", state: "done", activity: [] } as unknown as Trail;
    const entries = [{ kind: "tool_result" as const, call_id: "c-1", tool: "Bash", summary: "go test ./...", at: "2026-09-18T10:00:05Z" }];
    renderRow(
      <MessageRow
        message={message({ author_kind: "agent", body: "All green." })}
        authorLogin="onik97"
        isOwn
        trailBlock={{ trail, turns: [{ kind: "turn", entries, running: false, from: "2026-09-18T10:00:00Z", until: "2026-09-18T10:00:05Z" }] }}
        onEdit={noop}
        onDelete={noop}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: /Worked for 5s/ }));
    expect(screen.getByText("go test ./...")).toBeInTheDocument();
    expect(screen.getByText("All green.")).toBeInTheDocument();
  });
});
