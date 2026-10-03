import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PRChip } from "@/components/ticket/PRChip";
import type { PRLink } from "@/models/Ticket";

const pr = (state: PRLink["state"]): PRLink => ({
  owner: "acme",
  repo: "app",
  number: 42,
  title: "Fix login",
  sha: "abc",
  state,
});

describe("PRChip", () => {
  it.each([
    ["open", "text-success"],
    ["merged", "text-merged"],
    ["closed", "text-destructive"],
  ] as const)("colors a %s pull request with %s", (state, color) => {
    render(<PRChip pr={pr(state)} />);
    expect(screen.getByText("#42")).toHaveClass(color);
    expect(screen.getByRole("link", { name: `#42 Fix login, ${state}` })).toHaveAttribute(
      "href",
      "https://github.com/acme/app/pull/42",
    );
  });
});
