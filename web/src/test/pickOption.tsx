import { screen } from "@testing-library/react";
import type { UserEvent } from "@testing-library/user-event";

// Drives the Radix-based ui/select in tests; replaces user.selectOptions, which only works on native <select>.
export const pickOption = async (
  user: UserEvent,
  comboboxName: string | RegExp,
  optionName: string | RegExp,
): Promise<void> => {
  await user.click(await screen.findByRole("combobox", { name: comboboxName }));
  await user.click(await screen.findByRole("option", { name: optionName }));
};
