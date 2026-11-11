import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { RunnerManagerTag } from "@/components/runner/RunnerManagerTag";
import type { ManagerState } from "@/models/Runner";

describe("RunnerManagerTag", () => {
  it.each([
    ["ready", "1Password"],
    ["token rejected", "1Password token rejected"],
    ["unreachable", "1Password unreachable"],
  ] as [ManagerState["state"], string][])("words a %s manager as %s", (state, words) => {
    render(<RunnerManagerTag manager={{ manager: "op", state }} />);

    expect(screen.getByText(words)).toBeInTheDocument();
  });
});
