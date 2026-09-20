import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ConflictBanner } from "@/components/doc/collab/ConflictBanner";

describe("ConflictBanner", () => {
  it("explains the unresolvable merge and offers both recovery paths", async () => {
    const user = userEvent.setup();
    const onKeepMine = vi.fn();
    const onTakeServer = vi.fn();
    render(<ConflictBanner onKeepMine={onKeepMine} onTakeServer={onTakeServer} />);

    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText(/cannot merge it automatically/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Keep my version" }));
    expect(onKeepMine).toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Use server version" }));
    expect(onTakeServer).toHaveBeenCalled();
  });
});
