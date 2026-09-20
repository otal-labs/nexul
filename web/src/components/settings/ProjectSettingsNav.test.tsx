import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import {
  isProjectSettingsSection,
  ProjectSettingsNav,
} from "@/components/settings/ProjectSettingsNav";

describe("ProjectSettingsNav", () => {
  it("renders one link per section, each pointing at its section param", () => {
    render(
      <MemoryRouter initialEntries={["/projects/p-1/settings"]}>
        <ProjectSettingsNav active="general" />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "General" })).toHaveAttribute(
      "href",
      "/projects/p-1/settings?section=general",
    );
    expect(screen.getByRole("link", { name: "Categories" })).toHaveAttribute(
      "href",
      "/projects/p-1/settings?section=categories",
    );
    expect(screen.getByRole("link", { name: "Repositories" })).toHaveAttribute(
      "href",
      "/projects/p-1/settings?section=repositories",
    );
    expect(screen.getByRole("link", { name: "Services" })).toHaveAttribute(
      "href",
      "/projects/p-1/settings?section=services",
    );
    expect(screen.getByRole("link", { name: "Board" })).toHaveAttribute(
      "href",
      "/projects/p-1/settings?section=board",
    );
    expect(screen.getByRole("link", { name: "T3 pairing" })).toHaveAttribute(
      "href",
      "/projects/p-1/settings?section=pairing",
    );
    expect(screen.getByRole("link", { name: "Danger zone" })).toHaveAttribute(
      "href",
      "/projects/p-1/settings?section=danger",
    );
  });

  it("marks only the active section", () => {
    render(
      <MemoryRouter>
        <ProjectSettingsNav active="categories" />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "Categories" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "General" })).not.toHaveAttribute("aria-current");
    expect(screen.getByRole("link", { name: "Danger zone" })).not.toHaveAttribute("aria-current");
  });
});

describe("isProjectSettingsSection", () => {
  it("accepts every known section", () => {
    expect(isProjectSettingsSection("general")).toBe(true);
    expect(isProjectSettingsSection("categories")).toBe(true);
    expect(isProjectSettingsSection("repositories")).toBe(true);
    expect(isProjectSettingsSection("services")).toBe(true);
    expect(isProjectSettingsSection("board")).toBe(true);
    expect(isProjectSettingsSection("pairing")).toBe(true);
    expect(isProjectSettingsSection("danger")).toBe(true);
  });

  it("rejects anything else", () => {
    expect(isProjectSettingsSection("nope")).toBe(false);
    expect(isProjectSettingsSection(null)).toBe(false);
    expect(isProjectSettingsSection(undefined)).toBe(false);
    expect(isProjectSettingsSection("")).toBe(false);
  });
});
