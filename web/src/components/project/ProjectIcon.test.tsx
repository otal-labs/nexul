import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ProjectIcon } from "@/components/project/ProjectIcon";

describe("ProjectIcon", () => {
  it("renders the named lucide icon when it's a known name", () => {
    const { container } = render(<ProjectIcon icon="Rocket" />);
    expect(container.querySelector("svg")).toBeInTheDocument();
  });

  it("falls back to the prefix's first letter when icon is unset", () => {
    const { getByText } = render(<ProjectIcon icon="" prefix="deploy" />);
    expect(getByText("D")).toBeInTheDocument();
  });

  it("falls back to a generic glyph when neither icon nor prefix is set", () => {
    const { container } = render(<ProjectIcon />);
    expect(container.querySelector("svg")).toBeInTheDocument();
  });

  it("ignores an unrecognized icon name and falls back", () => {
    const { getByText } = render(<ProjectIcon icon="NotARealIcon" prefix="be" />);
    expect(getByText("B")).toBeInTheDocument();
  });
});
