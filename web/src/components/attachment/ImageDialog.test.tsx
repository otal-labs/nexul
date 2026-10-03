import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { ImageDialog } from "@/components/attachment/ImageDialog";

describe("ImageDialog", () => {
  it("opens the image in a dialog and copies its absolute link", async () => {
    const user = userEvent.setup();
    render(<ImageDialog src="blob:http://localhost/abc" alt="screenshot" link="/api/attachments/att-1" />);

    await user.click(screen.getByRole("button", { name: "Open screenshot" }));
    const dialog = await screen.findByRole("dialog", { name: "screenshot" });
    await user.click(screen.getByRole("button", { name: "Copy link" }));

    expect(dialog).toBeInTheDocument();
    await expect(navigator.clipboard.readText()).resolves.toBe(`${window.location.origin}/api/attachments/att-1`);
  });
});
