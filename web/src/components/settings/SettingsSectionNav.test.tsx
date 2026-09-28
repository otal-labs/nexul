import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { SettingsSectionNav } from "@/components/settings/SettingsSectionNav";

describe("SettingsSectionNav", () => {
  it("labels a group once, before its first item", () => {
    render(
      <MemoryRouter initialEntries={["/x"]}>
        <SettingsSectionNav
          ariaLabel="Sections"
          active="b"
          items={[
            { section: "a", label: "A", group: "First" },
            { section: "b", label: "B", group: "First" },
            { section: "c", label: "C", group: "Second" },
          ]}
        />
      </MemoryRouter>,
    );
    expect(screen.getAllByRole("listitem").map((item) => item.textContent)).toEqual(["First", "A", "B", "Second", "C"]);
    expect(screen.getByRole("link", { name: "B" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "C" })).toHaveAttribute("href", "/x?section=c");
  });

  it("renders no labels for ungrouped items", () => {
    render(
      <MemoryRouter>
        <SettingsSectionNav ariaLabel="Sections" active="a" items={[{ section: "a", label: "A" }]} />
      </MemoryRouter>,
    );
    expect(screen.getAllByRole("listitem")).toHaveLength(1);
  });
});
