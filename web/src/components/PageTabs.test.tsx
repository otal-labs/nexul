import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { describe, expect, it } from "vitest";

import { PageTabs, PageTabsContent, type PageTab } from "@/components/PageTabs";

const LocationProbe = () => {
  const location = useLocation();
  return <output aria-label="location">{location.pathname + location.search}</output>;
};

const renderTabs = (route: string, tabs: PageTab[]) =>
  render(
    <MemoryRouter initialEntries={[route]}>
      <Routes>
        <Route
          path="/page/:tab?"
          element={
            <PageTabs label="Example sections" tabs={tabs}>
              <PageTabsContent value="first">First body</PageTabsContent>
              <PageTabsContent value="second">Second body</PageTabsContent>
              <PageTabsContent value="secret">Secret body</PageTabsContent>
            </PageTabs>
          }
        />
      </Routes>
      <LocationProbe />
    </MemoryRouter>,
  );

const tabs: PageTab[] = [
  { value: "first", label: "First" },
  { value: "second", label: "Second" },
];

const location = () => screen.getByLabelText("location").textContent;

describe("PageTabs", () => {
  it("falls back to the first tab when the path names an unknown tab", () => {
    renderTabs("/page/nope", tabs);

    expect(screen.getByRole("tab", { name: "First" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("First body")).toBeInTheDocument();
    expect(screen.queryByText("Second body")).not.toBeInTheDocument();
  });

  it("never selects a hidden tab, even when the path asks for it", () => {
    renderTabs("/page/secret", [...tabs, { value: "secret", label: "Secret", hidden: true }]);

    expect(screen.queryByRole("tab", { name: "Secret" })).not.toBeInTheDocument();
    expect(screen.queryByText("Secret body")).not.toBeInTheDocument();
    expect(screen.getByText("First body")).toBeInTheDocument();
  });

  it("drops the tab list when only one tab is visible", () => {
    renderTabs("/page", [tabs[0]!, { value: "second", label: "Second", hidden: true }]);

    expect(screen.queryByRole("tablist")).not.toBeInTheDocument();
    expect(screen.getByText("First body")).toBeInTheDocument();
  });

  it("opens on the first tab when the path has no tab segment", () => {
    renderTabs("/page", tabs);

    expect(screen.getByRole("tablist", { name: "Example sections" })).toBeInTheDocument();
    expect(screen.getByText("First body")).toBeInTheDocument();
  });

  it("opens on the tab named by the path's last segment", () => {
    renderTabs("/page/second", tabs);

    expect(screen.getByRole("tab", { name: "Second" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Second body")).toBeInTheDocument();
  });

  it("navigates by path when a tab is clicked, keeping the query", async () => {
    const user = userEvent.setup();
    renderTabs("/page?setup=c1", tabs);

    await user.click(screen.getByRole("tab", { name: "Second" }));

    expect(screen.getByText("Second body")).toBeInTheDocument();
    expect(location()).toBe("/page/second?setup=c1");
  });

  it("swaps the tab segment instead of stacking another, and the first tab drops it", async () => {
    const user = userEvent.setup();
    renderTabs("/page/second", tabs);

    await user.click(screen.getByRole("tab", { name: "First" }));

    expect(location()).toBe("/page");
  });
});
