import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import {
  isProjectSettingsSection,
  ProjectSettingsNav,
} from "@/components/settings/ProjectSettingsNav";
import type { Project } from "@/models/Project";

const project: Project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, workspace_id: "ws-1", icon: "", tests_location: "same", setup: { finished: true, steps: {} }, created_at: "", updated_at: "" };

const renderNav = (canManageAccess: boolean) =>
  render(
    <MemoryRouter initialEntries={["/acme/projects/p-1/settings"]}>
      <ProjectSettingsNav project={project} active="general" canManageAccess={canManageAccess} />
    </MemoryRouter>,
  );

describe("ProjectSettingsNav", () => {
  it("renders one link per section under its group, each pointing at its section param", () => {
    renderNav(true);

    const sections: [string, string][] = [
      ["General", "general"],
      ["Access", "access"],
      ["Statuses", "statuses"],
      ["Categories", "categories"],
      ["Ticket types", "types"],
      ["Labels", "labels"],
      ["Repositories", "repositories"],
      ["Services", "services"],
      ["Danger zone", "danger"],
    ];
    for (const [name, section] of sections) {
      expect(screen.getByRole("link", { name })).toHaveAttribute("href", `/acme/projects/BE/settings/${section}`);
    }
    expect(screen.getByText("Board")).toBeInTheDocument();
    expect(screen.getByText("Delivery")).toBeInTheDocument();
  });

  it("leaves Access out for someone who can't change who gets in", () => {
    renderNav(false);
    expect(screen.queryByRole("link", { name: "Access" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "General" })).toBeInTheDocument();
  });
});

describe("isProjectSettingsSection", () => {
  it("accepts a known section and rejects anything else", () => {
    expect(isProjectSettingsSection("labels")).toBe(true);
    for (const section of ["board", "nope", "pairing", null, undefined, ""]) expect(isProjectSettingsSection(section)).toBe(false);
  });
});
