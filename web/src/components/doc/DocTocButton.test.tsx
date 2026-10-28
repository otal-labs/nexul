import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { DocTocButton } from "@/components/doc/DocTocButton";
import type { DocHeading } from "@/components/doc/docHeadings";

const headings: DocHeading[] = [
  { id: "severity", text: "Severity", level: 2 },
  { id: "declare", text: "Declare", level: 2 },
];

describe("DocTocButton", () => {
  it("renders nothing for a doc with fewer than two headings", () => {
    const { container } = render(<DocTocButton headings={headings.slice(0, 1)} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("opens the outline in a popover and closes it after a jump", async () => {
    const user = userEvent.setup();
    render(<DocTocButton headings={headings} />);

    await user.click(screen.getByRole("button", { name: "On this page" }));
    await user.click(await screen.findByRole("link", { name: "Declare" }));

    expect(screen.queryByRole("link", { name: "Declare" })).not.toBeInTheDocument();
  });
});
