import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import {
  isSettingsSection,
  SETTINGS_SECTIONS,
  SettingsNav,
  visibleSettingsSections,
  type SettingsVisibility,
} from "@/components/settings/SettingsNav";

const nothing: SettingsVisibility = {
  instanceSections: [],
  teamIsInstanceWide: false,
  showRoles: false,
  showPlays: false,
  showInterviewTemplate: false,
  showMentionLayout: false,
  showTeam: false,
};

const everything: SettingsVisibility = {
  instanceSections: ["instance", "sign-in", "connectors", "dns"],
  teamIsInstanceWide: true,
  showRoles: true,
  showPlays: true,
  showInterviewTemplate: true,
  showMentionLayout: true,
  showTeam: true,
};

const renderNav = (visibility: SettingsVisibility, active = "danger" as const) =>
  render(
    <MemoryRouter initialEntries={["/configuration"]}>
      <SettingsNav active={active} sections={visibleSettingsSections(visibility)} teamIsInstanceWide={visibility.teamIsInstanceWide} />
    </MemoryRouter>,
  );

describe("SettingsNav", () => {
  it("renders every section in two labelled groups for a viewer holding every permission", () => {
    renderNav(everything);
    const nav = within(screen.getByRole("navigation", { name: "Configuration sections" }));
    expect(nav.getAllByRole("listitem").map((item) => item.textContent)).toEqual([
      "This workspace",
      "Roles",
      "Plays",
      "Interview template",
      "Mention chips",
      "Danger zone",
      "Whole instance",
      "Instance",
      "Team",
      "Sign-in providers",
      "Connectors",
      "DNS",
    ]);
    expect(nav.getByRole("link", { name: "Team" })).toHaveAttribute("href", "/configuration/team");
    expect(nav.getByRole("link", { name: "Sign-in providers" })).toHaveAttribute("href", "/configuration/sign-in");
  });

  it("drops the whole-instance group and its label for a viewer holding none of its permissions", () => {
    renderNav({ ...everything, instanceSections: [], teamIsInstanceWide: false, showTeam: false });
    expect(screen.queryByText("Whole instance")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Instance" })).not.toBeInTheDocument();
    expect(screen.getByText("This workspace")).toBeInTheDocument();
  });

  it("puts Team with the workspace sections, ahead of Danger zone, for a member manager without accounts:read", () => {
    renderNav({ ...nothing, showRoles: true, showTeam: true });
    const nav = within(screen.getByRole("navigation", { name: "Configuration sections" }));
    expect(nav.getAllByRole("listitem").map((item) => item.textContent)).toEqual(["This workspace", "Roles", "Team", "Danger zone"]);
    expect(nav.getByRole("link", { name: "Team" })).toHaveAttribute("href", "/configuration/team");
  });

  it("hides every gated workspace section until its flag is set, leaving Danger zone", () => {
    renderNav(nothing);
    expect(screen.getAllByRole("link").map((link) => link.textContent)).toEqual(["Danger zone"]);
  });

  it("marks only the active section", () => {
    renderNav(everything, "danger");
    expect(screen.getByRole("link", { name: "Danger zone" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Roles" })).not.toHaveAttribute("aria-current");
  });
});

describe("visibleSettingsSections", () => {
  it("keeps nav order", () => {
    expect(visibleSettingsSections(everything)).toEqual([...SETTINGS_SECTIONS]);
  });
});

describe("isSettingsSection", () => {
  it("accepts every known section and rejects the personal ones that moved out", () => {
    for (const section of SETTINGS_SECTIONS) expect(isSettingsSection(section)).toBe(true);
    for (const section of ["appearance", "tokens", "pairing", "nope", "", null, undefined]) {
      expect(isSettingsSection(section)).toBe(false);
    }
  });
});
