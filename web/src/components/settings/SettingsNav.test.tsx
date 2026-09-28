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
  isInstanceAdmin: false,
  showRoles: false,
  showPlays: false,
  showInterviewTemplate: false,
  showMembers: false,
  showMentionLayout: false,
};

const everything: SettingsVisibility = {
  isInstanceAdmin: true,
  showRoles: true,
  showPlays: true,
  showInterviewTemplate: true,
  showMembers: true,
  showMentionLayout: true,
};

const renderNav = (visibility: SettingsVisibility, active = "danger" as const) =>
  render(
    <MemoryRouter initialEntries={["/configuration"]}>
      <SettingsNav active={active} sections={visibleSettingsSections(visibility)} />
    </MemoryRouter>,
  );

describe("SettingsNav", () => {
  it("renders every section in two labelled groups for an admin holding every permission", () => {
    renderNav(everything);
    const nav = within(screen.getByRole("navigation", { name: "Configuration sections" }));
    expect(nav.getAllByRole("listitem").map((item) => item.textContent)).toEqual([
      "This workspace",
      "Roles",
      "Plays",
      "Interview template",
      "Members",
      "Mention chips",
      "Danger zone",
      "Whole instance",
      "Instance",
      "Sign-in providers",
      "Connectors",
      "DNS",
      "Registered accounts",
    ]);
    expect(nav.getByRole("link", { name: "Members" })).toHaveAttribute("href", "/configuration?section=members");
    expect(nav.getByRole("link", { name: "Sign-in providers" })).toHaveAttribute("href", "/configuration?section=sign-in");
  });

  it("drops the whole-instance group and its label for a non-admin", () => {
    renderNav({ ...everything, isInstanceAdmin: false });
    expect(screen.queryByText("Whole instance")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Instance" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Registered accounts" })).not.toBeInTheDocument();
    expect(screen.getByText("This workspace")).toBeInTheDocument();
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
