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
  showInterviewTemplate: false,
  showMentionLayout: false,
  showTeam: false,
};

const everything: SettingsVisibility = {
  instanceSections: ["instance", "sign-in", "connectors", "dns", "templates"],
  teamIsInstanceWide: true,
  showGeneral: true,
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
      "Interview template",
      "Mention chips",
      "Danger zone",
    ]);
    expect(nav.getByRole("link", { name: "General" })).toHaveAttribute("href", "/acme/configuration/general");
    expect(nav.queryByRole("link", { name: "Plays" })).not.toBeInTheDocument();
  });

  it("puts Team with the workspace sections, ahead of Danger zone, for a member manager without accounts:read", () => {
    renderNav(workspaceSections({ ...nothing, showTeam: true }));
    const nav = within(screen.getByRole("navigation", { name: "Configuration sections" }));
    expect(nav.getAllByRole("listitem").map((item) => item.textContent)).toEqual(["Team", "Danger zone"]);
    expect(nav.getByRole("link", { name: "Team" })).toHaveAttribute("href", "/acme/configuration/team");
  });

  it("sets Danger zone apart with a hairline in the top row, and lets every label hold its medium width", () => {
    renderNav(workspaceSections(everything));
    const danger = screen.getByRole("link", { name: "Danger zone" }).closest("li");
    expect(danger?.previousElementSibling).toHaveAttribute("aria-hidden");
    expect(screen.getByText("Mention chips")).toHaveAttribute("data-label", "Mention chips");
  });

  it("hides every gated workspace section until its flag is set, leaving Danger zone", () => {
    renderNav(workspaceSections(nothing));
    expect(screen.getAllByRole("link").map((link) => link.textContent)).toEqual(["Danger zone"]);
  });
});

describe("isInstanceSection", () => {
  it("sends Team to Settings only for an accounts:read holder", () => {
    expect(isInstanceSection("team", true)).toBe(true);
    expect(isInstanceSection("team", false)).toBe(false);
    expect(isInstanceSection("connectors", false)).toBe(true);
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
