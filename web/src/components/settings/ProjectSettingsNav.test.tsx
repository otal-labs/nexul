import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import {
  isProjectSettingsSection,
  ProjectSettingsNav,
} from "@/components/settings/ProjectSettingsNav";
import type { Project } from "@/models/Project";

const project: Project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, icon: "", tests_location: "same", created_at: "", updated_at: "" };

describe("ProjectSettingsNav", () => {
  it("renders one link per section, each pointing at its section param", () => {
    render(
      <MemoryRouter initialEntries={["/acme/projects/p-1/settings"]}>
        <ProjectSettingsNav project={project} active="general" />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "General" })).toHaveAttribute(
      "href",
      "/acme/projects/BE/settings/general",
    );
    expect(screen.getByRole("link", { name: "Categories" })).toHaveAttribute(
      "href",
      "/acme/projects/BE/settings/categories",
    );
    expect(screen.getByRole("link", { name: "Repositories" })).toHaveAttribute(
      "href",
      "/acme/projects/BE/settings/repositories",
    );
    expect(screen.getByRole("link", { name: "Services" })).toHaveAttribute(
      "href",
      "/acme/projects/BE/settings/services",
    );
    expect(screen.getByRole("link", { name: "Board" })).toHaveAttribute(
      "href",
      "/acme/projects/BE/settings/board",
    );
    expect(screen.getByRole("link", { name: "Danger zone" })).toHaveAttribute(
      "href",
      "/acme/projects/BE/settings/danger",
    );
  });

  it("marks only the active section", () => {
    render(
      <MemoryRouter>
        <ProjectSettingsNav project={project} active="categories" />
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
    expect(isProjectSettingsSection("danger")).toBe(true);
  });

  it("rejects anything else", () => {
    expect(isProjectSettingsSection("nope")).toBe(false);
    expect(isProjectSettingsSection("pairing")).toBe(false);
    expect(isProjectSettingsSection(null)).toBe(false);
    expect(isProjectSettingsSection(undefined)).toBe(false);
    expect(isProjectSettingsSection("")).toBe(false);
  });
});
