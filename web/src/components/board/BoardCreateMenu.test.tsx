import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { BoardCreateMenu } from "@/components/board/BoardCreateMenu";

describe("BoardCreateMenu", () => {
  it("fires onNewTicket from the menu and closes it", async () => {
    const user = userEvent.setup();
    const onNewTicket = vi.fn();
    render(<BoardCreateMenu onNewTicket={onNewTicket} onNewCategory={() => {}} />);

    await user.click(screen.getByRole("button", { name: "Add" }));
    await user.click(await screen.findByRole("button", { name: "New ticket" }));

    expect(onNewTicket).toHaveBeenCalledOnce();
    expect(screen.queryByRole("button", { name: "New ticket" })).not.toBeInTheDocument();
  });

  it("fires onNewCategory from the menu", async () => {
    const user = userEvent.setup();
    const onNewCategory = vi.fn();
    render(<BoardCreateMenu onNewTicket={() => {}} onNewCategory={onNewCategory} />);

    await user.click(screen.getByRole("button", { name: "Add" }));
    await user.click(await screen.findByRole("button", { name: "New category" }));

    expect(onNewCategory).toHaveBeenCalledOnce();
  });
});
