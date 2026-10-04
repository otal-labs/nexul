import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

describe("Popover inside a dialog", () => {
  it("lets the mouse wheel scroll its list", async () => {
    render(
      <Dialog open>
        <DialogContent aria-describedby={undefined}>
          <DialogTitle>Report a bug</DialogTitle>
          <Popover open>
            <PopoverTrigger>Found in…</PopoverTrigger>
            <PopoverContent>
              <div data-testid="list" style={{ overflowY: "auto" }}>
                <button type="button">NEXUL-1</button>
              </div>
            </PopoverContent>
          </Popover>
        </DialogContent>
      </Dialog>,
    );
    const list = screen.getByTestId("list");
    Object.defineProperty(list, "scrollHeight", { value: 600 });
    Object.defineProperty(list, "clientHeight", { value: 200 });

    const wheel = fireEvent.wheel(await screen.findByRole("button", { name: "NEXUL-1" }), { deltaY: 100 });

    expect(wheel).toBe(true);
  });
});
