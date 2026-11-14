import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Logo } from "@/components/Logo";
import { BrandingEffect } from "@/ee/components/BrandingEffect";
import { useThemeStore } from "@/stores/themeStore";
import { RunsOnNexul } from "@/ee/components/RunsOnNexul";
import { getLookKey } from "@/ee/hooks/BrandingHooks";
import { useProductName } from "@/ee/hooks/useProductName";
import type { Look } from "@/ee/models/Branding";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: String }));

const look = (over: Partial<Look> = {}): Look => ({
  enabled: true,
  product_name: "Acme Hub",
  logo_url: "/api/branding/logo?v=v1",
  light: { themed: false, roles: { brand: "oklch(0.5 0.2 250)" } },
  dark: { themed: true, roles: { background: "oklch(0.2 0.03 260)", brand: "oklch(0.7 0.15 150)" } },
  css: "",
  version: "x",
  ...over,
});

const Name = () => <p>{useProductName()}</p>;

const renderWithLook = (data: Look) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getLookKey], data);
  const view = render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <BrandingEffect />
        <Name />
        <Logo />
        <RunsOnNexul />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { client, ...view };
};

beforeEach(() => {
  document.head.innerHTML = '<link rel="icon" href="/favicon.svg" /><link rel="apple-touch-icon" href="/apple.png" />';
  document.documentElement.removeAttribute("style");
  document.documentElement.classList.add("dark");
  document.title = "Nexul";
  useThemeStore.setState({ theme: "dark", themeId: "console" });
});

describe("the instance's own look", () => {
  it("puts the name, the mark, the icon and the colours everywhere Nexul's were, and says what it runs on", async () => {
    renderWithLook(look());

    expect(screen.getByText("Acme Hub")).toBeInTheDocument();
    expect(screen.getByText("Runs on Nexul")).toHaveAttribute("href", "https://nexul.io");
    expect(document.querySelector<HTMLImageElement>("img")?.getAttribute("src")).toBe("/api/branding/logo?v=v1");
    await waitFor(() => expect(document.title).toBe("Acme Hub"));
    expect(document.head.querySelector('link[rel="icon"]')?.getAttribute("href")).toBe("/api/branding/logo?v=v1");
    expect(document.head.querySelectorAll('link[rel="icon"], link[rel="apple-touch-icon"]')).toHaveLength(1);
    expect(document.documentElement.style.getPropertyValue("--background")).toBe("oklch(0.2 0.03 260)");
  });

  it("goes back to Nexul's name, mark, icons and colours once the look no longer shows", async () => {
    const { client } = renderWithLook(look());
    await waitFor(() => expect(document.title).toBe("Acme Hub"));

    client.setQueryData([getLookKey], look({ enabled: false, product_name: "", logo_url: "" }));

    expect(await screen.findByText("Nexul")).toBeInTheDocument();
    await waitFor(() => expect(document.title).toBe("Nexul"));
    expect(screen.queryByText("Runs on Nexul")).not.toBeInTheDocument();
    expect(document.querySelector("img")).toBeNull();
    expect(document.head.querySelector('link[rel="icon"]')?.getAttribute("href")).toBe("/favicon.svg");
    expect(document.head.querySelector('link[rel="apple-touch-icon"]')).not.toBeNull();
    expect(document.documentElement.style.getPropertyValue("--background")).toBe("");
  });

  it("adds the instance's own CSS last in the head, follows a change, and drops it once the look no longer shows", async () => {
    const { client } = renderWithLook(look({ css: "[data-app-sidebar]{font-family:monospace}" }));
    await waitFor(() => expect(document.getElementById("branding-css")?.textContent).toBe("[data-app-sidebar]{font-family:monospace}"));
    expect(document.head.lastElementChild?.id).toBe("branding-css");

    client.setQueryData([getLookKey], look({ css: ".app-frame{border-radius:0}" }));
    await waitFor(() => expect(document.getElementById("branding-css")?.textContent).toBe(".app-frame{border-radius:0}"));
    expect(document.querySelectorAll("#branding-css")).toHaveLength(1);

    client.setQueryData([getLookKey], look({ enabled: false, css: ".app-frame{border-radius:0}" }));
    await waitFor(() => expect(document.getElementById("branding-css")).toBeNull());
  });

  it("leaves the unread count in the tab title", async () => {
    document.title = "(3) Nexul";
    renderWithLook(look());
    await waitFor(() => expect(document.head.querySelector('link[rel="icon"]')?.getAttribute("href")).toBe("/api/branding/logo?v=v1"));
    expect(document.title).toBe("(3) Nexul");
  });
});
