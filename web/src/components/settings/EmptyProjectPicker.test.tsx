import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { EmptyProjectPicker } from "@/components/settings/EmptyProjectPicker";

const notes = { id: "t3-notes", title: "notes", path: "/home/sam/Code/notes" };
const api = { id: "t3-api", title: "api", path: "/home/sam/Code/api", repository: "github.com/acme/api" };

describe("EmptyProjectPicker", () => {
  it("picks a project with no git remote, and offers none that has one", async () => {
    const onPick = vi.fn();
    const user = userEvent.setup();
    render(<EmptyProjectPicker computerName="laptop" projects={[notes, api]} onPick={onPick} />);
    await user.click(screen.getByRole("combobox", { name: "Use an empty project on laptop" }));
    expect(screen.queryByRole("option", { name: /api/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole("option", { name: /notes/ }));
    expect(onPick).toHaveBeenCalledWith("t3-notes");
  });

  it("shows nothing when the computer has no empty project", () => {
    render(<EmptyProjectPicker computerName="laptop" projects={[api]} onPick={vi.fn()} />);
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
  });
});
