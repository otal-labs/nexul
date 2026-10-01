import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { RowActions } from "@/components/listpane/RowActions";

describe("RowActions", () => {
  it("lists the places to move to with the current one checked, and moves on a pick", async () => {
    const user = userEvent.setup();
    const onMove = vi.fn();
    const options = [
      { id: "f-main", name: "Main" },
      { id: "f-gs", name: "GetSource" },
    ];
    render(<RowActions itemLabel="EP01" moveTo={{ label: "Move to folder", options, currentId: "f-main", onMove }} />);

    await user.click(screen.getByRole("button", { name: "More actions for EP01" }));
    await user.click(screen.getByRole("menuitem", { name: "Move to folder" }));
    expect(await screen.findByRole("menuitemradio", { name: "Main" })).toHaveAttribute("aria-checked", "true");
    expect(screen.getByRole("menuitemradio", { name: "GetSource" })).toHaveAttribute("aria-checked", "false");

    screen.getByRole("menuitemradio", { name: "GetSource" }).focus();
    await user.keyboard("{Enter}");
    expect(onMove).toHaveBeenCalledWith("f-gs");
  });
});
