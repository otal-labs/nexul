import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { LinkGroupSection } from "@/components/ticket/LinkGroupSection";

const group = (count: number) => (
  <LinkGroupSection title="Blocks">
    {Array.from({ length: count }, (_, i) => (
      <li key={i}>{`CLTCH-${i}`}</li>
    ))}
  </LinkGroupSection>
);

describe("LinkGroupSection", () => {
  it("shows five rows and the rest behind Show N more", async () => {
    const user = userEvent.setup();
    render(group(9));
    expect(screen.getAllByRole("listitem")).toHaveLength(5);

    await user.click(screen.getByRole("button", { name: "Show 4 more" }));
    expect(screen.getAllByRole("listitem")).toHaveLength(9);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("shows a sixth row rather than a button for one", () => {
    render(group(6));
    expect(screen.getAllByRole("listitem")).toHaveLength(6);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
