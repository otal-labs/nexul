import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it } from "vitest";

import { TabSegmentContext, useTabPath } from "@/hooks/useTabPath";

const Probe = () => {
  const { current, tabPath } = useTabPath();
  return <p>{`${current ?? "default"} ${tabPath("projects")} ${tabPath()}`}</p>;
};

const renderAt = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/:ws/settings/:section/:tab?" element={<Probe />} />
        <Route path="/:ws/access" element={<Probe />} />
        <Route path="/:ws/access/roles/:roleId?" element={<TabSegmentContext value="roles"><Probe /></TabSegmentContext>} />
      </Routes>
    </MemoryRouter>,
  );

describe("useTabPath", () => {
  it.each([
    ["a tab ending the path", "/acme/settings/team/invitations", "invitations /acme/settings/team/projects /acme/settings/team"],
    ["the default tab, with no segment", "/acme/access", "default /acme/access/projects /acme/access"],
    ["a record opened inside a static tab", "/acme/access/roles/r-dev", "roles /acme/access/projects /acme/access"],
    ["a static tab with no record open", "/acme/access/roles/", "roles /acme/access/projects /acme/access"],
  ])("builds tab links from the path before the tab's segment: %s", (_name, path, expected) => {
    renderAt(path);
    expect(screen.getByText(expected)).toBeInTheDocument();
  });
});
