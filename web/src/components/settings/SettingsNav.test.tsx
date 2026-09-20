import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { isSettingsSection, SettingsNav } from "@/components/settings/SettingsNav";

const renderNav = (overrides: Partial<Parameters<typeof SettingsNav>[0]> = {}) =>
  render(
    <MemoryRouter initialEntries={["/settings"]}>
      <SettingsNav
        active="instance"
        showInstanceAccess={false}
        showRoles={false}
        showPlays={false}
        showMentionLayout={false}
        {...overrides}
      />
    </MemoryRouter>,
  );

describe("SettingsNav", () => {
  it("renders one link per ungated section, each pointing at its section param", () => {
    renderNav();
    expect(screen.getByRole("link", { name: "Instance" })).toHaveAttribute("href", "/settings?section=instance");
    expect(screen.getByRole("link", { name: "Appearance" })).toHaveAttribute("href", "/settings?section=appearance");
    expect(screen.getByRole("link", { name: "Tokens" })).toHaveAttribute("href", "/settings?section=tokens");
    expect(screen.getByRole("link", { name: "T3 pairing" })).toHaveAttribute("href", "/settings?section=pairing");
    expect(screen.getByRole("link", { name: "Connectors" })).toHaveAttribute("href", "/settings?section=connectors");
    expect(screen.getByRole("link", { name: "Danger zone" })).toHaveAttribute("href", "/settings?section=danger");
  });

  it("marks only the active section", () => {
    renderNav({ active: "tokens" });
    expect(screen.getByRole("link", { name: "Tokens" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Instance" })).not.toHaveAttribute("aria-current");
  });

  it("hides gated sections until their flag is set", () => {
    renderNav();
    expect(screen.queryByRole("link", { name: "Registered accounts" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Roles" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Plays" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Mention chips" })).not.toBeInTheDocument();
  });

  it("shows gated sections when their flag is set", () => {
    renderNav({ showInstanceAccess: true, showRoles: true, showPlays: true, showMentionLayout: true });
    expect(screen.getByRole("link", { name: "Registered accounts" })).toHaveAttribute("href", "/settings?section=access");
    expect(screen.getByRole("link", { name: "Roles" })).toHaveAttribute("href", "/settings?section=roles");
    expect(screen.getByRole("link", { name: "Plays" })).toHaveAttribute("href", "/settings?section=plays");
    expect(screen.getByRole("link", { name: "Mention chips" })).toHaveAttribute("href", "/settings?section=mentions");
  });
});

describe("isSettingsSection", () => {
  it("accepts every known section", () => {
    for (const section of ["instance", "roles", "plays", "mentions", "appearance", "tokens", "pairing", "connectors", "access", "danger"]) {
      expect(isSettingsSection(section)).toBe(true);
    }
  });

  it("rejects anything else", () => {
    expect(isSettingsSection("nope")).toBe(false);
    expect(isSettingsSection(null)).toBe(false);
    expect(isSettingsSection(undefined)).toBe(false);
    expect(isSettingsSection("")).toBe(false);
  });
});
