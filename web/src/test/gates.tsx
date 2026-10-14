import { screen } from "@testing-library/react";
import { expect } from "vitest";

import { settle } from "@/test/viewer";

export interface WriteControl {
  role: "button" | "menuitem" | "switch";
  name: string | RegExp;
  // The sentence a locked control gives: "Your role can't <verb>".
  reason: string;
  // A switch whose whole row is left out for a reader, rather than shown disabled.
  rowHidden?: boolean;
}

// A reader's write controls are absent (a switch stays, disabled), or locked with the reason when the switch is on.
export const expectWriteControls = async (controls: WriteControl[], showLocked: boolean) => {
  await settle();
  for (const { role, name, reason, rowHidden } of controls) {
    if (!showLocked && (role !== "switch" || rowHidden)) expect(screen.queryByRole(role, { name })).not.toBeInTheDocument();
    if (!showLocked && role === "switch" && !rowHidden) expect(await screen.findByRole(role, { name })).toBeDisabled();
    if (showLocked) {
      const control = await screen.findByRole(role, { name });
      expect(control).toHaveAttribute("aria-disabled", "true");
      expect(control).toHaveAccessibleDescription(reason);
    }
  }
};
