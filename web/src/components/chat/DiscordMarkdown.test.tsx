import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";

import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";

const renderMarkdown = (text: string, mentionHandles?: string[]) =>
  render(
    <MemoryRouter>
      <DiscordMarkdown text={text} {...(mentionHandles ? { mentionHandles } : {})} />
    </MemoryRouter>,
  );

describe("DiscordMarkdown", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders bold, italics, and inline code", () => {
    const { container } = renderMarkdown("Deploy of **nexul-web** on `prod` was *slow* and _late_");
    expect(container.querySelector("strong")).toHaveTextContent("nexul-web");
    expect(container.querySelector("code")).toHaveTextContent("prod");
    expect([...container.querySelectorAll("em")].map((e) => e.textContent)).toEqual(["slow", "late"]);
  });

  it("keeps markers inside code, URLs, and snake_case words as typed", () => {
    const { container } = renderMarkdown("`**not bold**` see https://example.com/a_b_c/x*y*z and some_table_name");
    expect(container.querySelector("code")).toHaveTextContent("**not bold**");
    expect(container.querySelector("em")).toBeNull();
    expect(container.querySelector("strong")).toBeNull();
    expect(screen.getByRole("link")).toHaveAttribute("href", "https://example.com/a_b_c/x*y*z");
    expect(container).toHaveTextContent("some_table_name");
  });

  it("links a masked http(s) link with formatted label and shows where it goes", () => {
    renderMarkdown("[`a1b2c3d`](https://example.com/commit/a1b2c3d) Fix the flaky test");
    const link = screen.getByRole("link", { name: "a1b2c3d" });
    expect(link).toHaveAttribute("href", "https://example.com/commit/a1b2c3d");
    expect(link).toHaveAttribute("title", "https://example.com/commit/a1b2c3d");
    expect(link.querySelector("code")).not.toBeNull();
  });

  it.each([
    ["javascript:", "[click me](javascript:alert(document.cookie))"],
    ["data:", "[click me](data:text/html;base64,PHNjcmlwdD4=)"],
    ["a relative path", "[click me](/api/attachments/secret)"],
    ["vbscript:", "[click me](vbscript:msgbox)"],
  ])("never links a masked %s URL; the source reads as text", (_name, text) => {
    const { container } = renderMarkdown(text);
    expect(container.querySelector("a")).toBeNull();
    expect(container).toHaveTextContent(text);
  });

  it("does not nest a bare URL inside a masked link's label", () => {
    const { container } = renderMarkdown("[https://example.com/a](https://example.com/b)");
    const links = container.querySelectorAll("a");
    expect(links).toHaveLength(1);
    expect(links[0]).toHaveAttribute("href", "https://example.com/b");
  });

  it("links a bare URL, leaving the sentence's punctuation outside it", () => {
    renderMarkdown("Notes: https://example.com/changelog.");
    expect(screen.getByRole("link")).toHaveAttribute("href", "https://example.com/changelog");
  });

  it("formats <t:…> codes as the reader's local time, the exact moment on the element", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-10-06T20:11:00Z"));
    const at = Date.UTC(2026, 9, 6, 20, 11) / 1000;
    const { container } = renderMarkdown(`Went offline <t:${at}:F>, back <t:${at - 7200}:R>`);
    const times = container.querySelectorAll("time");
    expect(times[0]).toHaveAttribute("dateTime", "2026-10-06T20:11:00.000Z");
    expect(times[0]).toHaveTextContent("2026");
    expect(times[1]).toHaveTextContent("2 hours ago");
    expect(container).not.toHaveTextContent("<t:");
  });

  it("renders list lines and quotes, and bolds only the handles the server parsed as mentions", () => {
    const { container } = renderMarkdown("**Labels:**\n - alertname = HighErrorRate\n> on call: @lena and @nobody", ["lena"]);
    expect(container).toHaveTextContent("•alertname = HighErrorRate");
    const strong = [...container.querySelectorAll("strong")].map((s) => s.textContent);
    expect(strong).toEqual(["Labels:", "@lena"]);
  });
});
