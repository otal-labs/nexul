import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { MessageRow } from "@/components/chat/MessageRow";
import type { Message } from "@/models/Chat";
import type { Embed } from "@/models/Embed";
import { unknownPerson } from "@/models/Person";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn() },
  errorMessage: vi.fn(() => "error"),
}));

const botMessage = (overrides: Partial<Message>): Message => ({
  id: "m1",
  conversation_id: "c1",
  author_id: "b-ci",
  author_kind: "bot",
  author_name: "CI",
  body: "",
  mentions: null,
  created_at: "2026-10-06T20:11:00Z",
  updated_at: "2026-10-06T20:11:00Z",
  ...overrides,
});

const noop = async () => {};

const renderBot = (message: Message, continuation = false) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <MessageRow message={message} author={unknownPerson(message.author_id)} isOwn={false} continuation={continuation} onEdit={noop} onDelete={noop} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const unbreakable = "9f2c7a4e1b8d3f6a0c5e7b9d2f4a6c8e".repeat(8);

const worstEmbeds = (): Embed[] => [
  {
    title: `Snapshot ${unbreakable} restored`,
    url: "https://example.com/snapshots/1",
    description: "x".repeat(4096),
    thumbnail: { url: "https://example.com/thumb.png" },
    fields: Array.from({ length: 25 }, (_, i) => ({ name: `public.table_${i + 1}`, value: i === 7 ? unbreakable : "matched", inline: i % 2 === 0 })),
    footer: { text: "backup-verifier 3.4.1" },
    timestamp: "2026-10-06T20:10:00Z",
  },
  ...Array.from({ length: 9 }, (_, i): Embed => ({ title: `Report part ${i + 2}` })),
];

const media = (url: string) => `/api/botwebhooks/media?message=m1&url=${encodeURIComponent(url)}`;

const mediaCalls = () =>
  vi
    .mocked(api.get)
    .mock.calls.map(([path]) => path)
    .filter((path) => path.startsWith("/api/botwebhooks/media"));

const imageSources = (container: HTMLElement) => [...container.querySelectorAll("img")].map((img) => img.getAttribute("src"));

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.get).mockResolvedValue({ data: new Blob(["png"]) });
  globalThis.URL.createObjectURL = vi.fn(() => "blob:media");
});

describe("MessageRow for a bot", () => {
  it("shows the snapshot name, the BOT tag, and only the reaction action, never edit or delete", () => {
    renderBot(botMessage({ body: "Deploy finished for **nexul-web**" }));
    expect(screen.getByText("CI")).toBeInTheDocument();
    expect(screen.getByText("Bot")).toBeInTheDocument();
    expect(screen.getByText("nexul-web", { selector: "strong" })).toBeInTheDocument();
    expect(screen.getByLabelText("Add reaction")).toBeInTheDocument();
    expect(screen.queryByLabelText("Edit message")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Delete message")).not.toBeInTheDocument();
    expect(screen.queryByText("b-ci")).not.toBeInTheDocument();
  });

  it("names the sending bot beside a post that renamed itself, so a fake 'GitHub' shows where it came from", () => {
    renderBot(botMessage({ author_name: "GitHub", via: "CI", body: "hi" }));
    expect(screen.getByText("via CI")).toBeInTheDocument();
  });

  it("drops the avatar and header when it continues the same bot's group", () => {
    renderBot(botMessage({ body: "second" }), true);
    expect(screen.queryByText("CI")).not.toBeInTheDocument();
    expect(screen.getByText("second")).toBeInTheDocument();
  });

  it.each([
    ["no avatar", undefined],
    ["a javascript: avatar", "javascript:alert(1)"],
  ])("shows the glyph for %s and asks nothing of the proxy", (_name, avatar) => {
    const { container } = renderBot(botMessage({ ...(avatar ? { author_avatar_url: avatar } : {}), body: "hi" }));
    expect(container.querySelector("img")).toHaveAttribute("src", "/favicon.svg");
    expect(mediaCalls()).toEqual([]);
  });

  it("loads a sender's avatar override through the media proxy, never from the sender's host", async () => {
    const { container } = renderBot(botMessage({ author_avatar_url: "https://example.com/ci.png", body: "hi" }));
    await vi.waitFor(() => expect(container.querySelector("img")).toHaveAttribute("src", "blob:media"));
    expect(api.get).toHaveBeenCalledWith(media("https://example.com/ci.png"), { responseType: "blob" });
  });

  it("falls back to the glyph and no embed image when the proxy refuses, never to the sender's URL", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("502"));
    const { container } = renderBot(
      botMessage({ author_avatar_url: "https://example.com/ci.png", embeds: [{ title: "Run", image: { url: "https://example.com/chart.png" } }] }),
    );
    await vi.waitFor(() => expect(mediaCalls()).toHaveLength(2));
    await act(async () => {});
    expect(imageSources(container)).toEqual(["/favicon.svg"]);
  });

  it("loads the bot's own served avatar through the authenticated client", async () => {
    const { container } = renderBot(botMessage({ author_avatar_url: "/api/botwebhooks/b-ci/avatar?v=2", body: "hi" }));
    await vi.waitFor(() => expect(container.querySelector("img")).toHaveAttribute("src", "blob:media"));
    expect(api.get).toHaveBeenCalledWith("/api/botwebhooks/b-ci/avatar?v=2", { responseType: "blob" });
  });

  it("folds the worst-case post: six of 25 fields, two of ten embeds, the long name truncated not dropped", async () => {
    const user = userEvent.setup();
    const name = "Nightly backup verifier for acme-production-eu-west-2-replica-database-cluster";
    renderBot(botMessage({ author_name: name, embeds: worstEmbeds() }));
    expect(screen.getByText(name)).toBeInTheDocument();
    expect(screen.getAllByRole("term")).toHaveLength(6);
    expect(screen.queryByText("Report part 3")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Show 19 more fields" }));
    expect(screen.getAllByRole("term")).toHaveLength(25);
    expect(screen.getByText(unbreakable)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Show less" })).toHaveAttribute("aria-expanded", "true");

    await user.click(screen.getByRole("button", { name: "Show 8 more embeds" }));
    expect(screen.getByText("Report part 10")).toBeInTheDocument();
  });

  it("links the title and author and loads images only over http(s)", async () => {
    const { container } = renderBot(
      botMessage({
        embeds: [
          { title: "Safe", url: "https://example.com/run/1", image: { url: "https://example.com/chart.png" }, author: { name: "alice", url: "https://example.com/alice" } },
          { author: { name: "bob", url: "javascript:alert(1)", icon_url: "javascript:alert(2)" }, title: "Unsafe", url: "javascript:alert(1)", image: { url: "data:image/svg+xml,<svg onload=alert(1)>" }, thumbnail: { url: "file:///etc/passwd" } },
        ],
      }),
    );
    expect(screen.getByRole("link", { name: "Safe" })).toHaveAttribute("href", "https://example.com/run/1");
    expect(screen.queryByRole("link", { name: "Unsafe" })).not.toBeInTheDocument();
    expect(screen.getByText("Unsafe")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "alice" })).toHaveAttribute("href", "https://example.com/alice");
    expect(screen.queryByRole("link", { name: "bob" })).not.toBeInTheDocument();
    await vi.waitFor(() => expect(imageSources(container)).toEqual(["/favicon.svg", "blob:media"]));
    expect(mediaCalls()).toEqual([media("https://example.com/chart.png")]);
  });

  it("asks the proxy for every picture an embed shows, on behalf of its message", async () => {
    const { container } = renderBot(
      botMessage({
        embeds: [
          {
            author: { name: "alice", icon_url: "https://example.com/alice.png" },
            title: "Run",
            image: { url: "https://example.com/chart.png" },
            thumbnail: { url: "https://example.com/thumb.png" },
            footer: { text: "CI", icon_url: "https://example.com/ci.png" },
          },
        ],
      }),
    );
    await vi.waitFor(() => expect(container.querySelectorAll("img[src='blob:media']")).toHaveLength(4));
    expect(mediaCalls().sort()).toEqual(
      ["https://example.com/alice.png", "https://example.com/chart.png", "https://example.com/ci.png", "https://example.com/thumb.png"].map(media),
    );
  });

  it("drops an embed's author line that only repeats the bot's name, and keeps one that links somewhere", () => {
    renderBot(
      botMessage({
        embeds: [
          { author: { name: "CI" }, title: "Build 1" },
          { author: { name: "CI", url: "https://example.com/ci" }, title: "Build 2" },
        ],
      }),
    );
    expect(screen.getAllByText("CI")).toHaveLength(2);
    expect(screen.getByRole("link", { name: "CI" })).toHaveAttribute("href", "https://example.com/ci");
  });

  it("shows the footer with the calendar time and the exact moment on hover", () => {
    renderBot(botMessage({ embeds: [{ title: "Run", footer: { text: "GitHub Actions" }, timestamp: "2026-10-06T20:10:00Z" }] }));
    const footer = screen.getByText("GitHub Actions").parentElement!;
    const time = within(footer).getByText((_, el) => el?.tagName === "TIME");
    expect(time).toHaveAttribute("dateTime", "2026-10-06T20:10:00Z");
    expect(time).toHaveAttribute("title", new Date("2026-10-06T20:10:00Z").toLocaleString());
  });

  it("reads a footer time without a zone as UTC, as Uptime Kuma sends it and Discord reads it", () => {
    vi.stubEnv("TZ", "Asia/Kolkata");
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date("2026-10-06T22:00:00Z"));
    try {
      renderBot(botMessage({ embeds: [{ title: "Down", footer: { text: "Uptime Kuma" }, timestamp: "2026-10-06 20:10:00.000" }] }));
      expect(screen.getByText("Today at 01:40")).toBeInTheDocument();
    } finally {
      vi.useRealTimers();
      vi.unstubAllEnvs();
    }
  });
});
