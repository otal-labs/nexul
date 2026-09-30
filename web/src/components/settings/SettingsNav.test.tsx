import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import {
  isInstanceSection,
  isSettingsSection,
  SETTINGS_SECTIONS,
  SettingsNav,
  visibleSettingsSections,
  type SettingsSection,
  type SettingsVisibility,
} from "@/components/settings/SettingsNav";

const nothing: SettingsVisibility = {
  instanceSections: [],
  teamIsInstanceWide: false,
  showGeneral: false,
  showRoles: false,
  showPlays: false,
  showInterviewTemplate: false,
  showMentionLayout: false,
  showTeam: false,
};

const everything: SettingsVisibility = {
  instanceSections: ["instance", "sign-in", "connectors", "dns"],
  teamIsInstanceWide: true,
  showGeneral: true,
  showRoles: true,
  showPlays: true,
  showInterviewTemplate: true,
  showMentionLayout: true,
  showTeam: true,
};

// What ConfigurationPage hands the nav: the visible sections with the instance ones taken out.
const workspaceSections = (visibility: SettingsVisibility) =>
  visibleSettingsSections(visibility).filter((section) => !isInstanceSection(section, visibility.teamIsInstanceWide));

const renderNav = (sections: SettingsSection[], active: SettingsSection = "danger") =>
  render(
    <MemoryRouter initialEntries={["/acme/configuration"]}>
      <SettingsNav active={active} sections={sections} />
    </MemoryRouter>,
  );

describe("SettingsNav", () => {
  it("lists the workspace sections under /configuration with no group label, and none of the instance ones", () => {
    renderNav(workspaceSections(everything));
    const nav = within(screen.getByRole("navigation", { name: "Configuration sections" }));
    expect(nav.getAllByRole("listitem").map((item) => item.textContent)).toEqual([
      "General",
      "Roles",
      "Plays",
      "Interview template",
      "Mention chips",
      "Danger zone",
    ]);
    expect(nav.getByRole("link", { name: "General" })).toHaveAttribute("href", "/acme/configuration/general");
    expect(nav.getByRole("link", { name: "Plays" })).toHaveAttribute("href", "/acme/configuration/plays");
  });

  it("puts Team with the workspace sections, ahead of Danger zone, for a member manager without accounts:read", () => {
    renderNav(workspaceSections({ ...nothing, showRoles: true, showTeam: true }));
    const nav = within(screen.getByRole("navigation", { name: "Configuration sections" }));
    expect(nav.getAllByRole("listitem").map((item) => item.textContent)).toEqual(["Roles", "Team", "Danger zone"]);
    expect(nav.getByRole("link", { name: "Team" })).toHaveAttribute("href", "/acme/configuration/team");
  });

  it("hides every gated workspace section until its flag is set, leaving Danger zone", () => {
    renderNav(workspaceSections(nothing));
    expect(screen.getAllByRole("link").map((link) => link.textContent)).toEqual(["Danger zone"]);
  });

  it("marks only the active section", () => {
    renderNav(workspaceSections(everything), "danger");
    expect(screen.getByRole("link", { name: "Danger zone" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Roles" })).not.toHaveAttribute("aria-current");
  });
});

describe("visibleSettingsSections", () => {
  it("keeps nav order", () => {
    expect(visibleSettingsSections(everything)).toEqual([...SETTINGS_SECTIONS]);
  });
});

describe("isInstanceSection", () => {
  it("sends Team to Settings only for an accounts:read holder", () => {
    expect(isInstanceSection("team", true)).toBe(true);
    expect(isInstanceSection("team", false)).toBe(false);
    expect(isInstanceSection("connectors", false)).toBe(true);
    expect(isInstanceSection("roles", true)).toBe(false);
  });
});

describe("isSettingsSection", () => {
  it("accepts every known section and rejects the personal ones", () => {
    for (const section of SETTINGS_SECTIONS) expect(isSettingsSection(section)).toBe(true);
    for (const section of ["appearance", "tokens", "pairing", "nope", "", null, undefined]) {
      expect(isSettingsSection(section)).toBe(false);
    }
  });
});
