import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PersonAvatar } from "@/components/PersonAvatar";

const fallback = (ui: React.ReactElement) => {
  const { container } = render(ui);
  fireEvent.error(container.querySelector("img")!);
};

describe("PersonAvatar", () => {
  it("takes a missing picture's initials from the display name, so two people sharing a first letter differ", () => {
    fallback(<PersonAvatar login="alice_nw" name="Alice Moreau" label="Alice Moreau" src="/a.png" />);
    expect(screen.getByTitle("Alice Moreau")).toHaveTextContent("AM");
  });

  it("falls back to the login's initial without a display name", () => {
    fallback(<PersonAvatar login="alice_nw" name="" src="/a.png" />);
    expect(screen.getByTitle("alice_nw")).toHaveTextContent(/^A$/);
  });

  it("paints the person's fill while the picture loads and drops it once the picture lands", () => {
    render(<PersonAvatar login="alice_nw" name="Alice Moreau" label="Alice Moreau" src="/a.png" />);
    const frame = screen.getByTitle("Alice Moreau").parentElement!;
    expect(frame.style.backgroundColor).not.toBe("");
    fireEvent.load(screen.getByTitle("Alice Moreau"));
    expect(frame.style.backgroundColor).toBe("");
  });
});
