import { act, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Trash2 } from "lucide-react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";

describe("ConfirmDestroyButton", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("does not call onConfirm on the first click — it arms instead", async () => {
    const onConfirm = vi.fn();
    const user = userEvent.setup();
    render(<ConfirmDestroyButton icon={Trash2} idleLabel="Revoke" onConfirm={onConfirm} />);

    await user.click(screen.getByRole("button", { name: "Revoke" }));

    expect(onConfirm).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Confirm" })).toBeInTheDocument();
  });

  it("calls onConfirm and reverts to idle when the confirm button is clicked", async () => {
    const onConfirm = vi.fn();
    const user = userEvent.setup();
    render(<ConfirmDestroyButton icon={Trash2} idleLabel="Revoke" onConfirm={onConfirm} />);

    await user.click(screen.getByRole("button", { name: "Revoke" }));
    await user.click(screen.getByRole("button", { name: "Confirm" }));

    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: "Revoke" })).toBeInTheDocument();
  });

  it("cancels back to idle without calling onConfirm", async () => {
    const onConfirm = vi.fn();
    const user = userEvent.setup();
    render(<ConfirmDestroyButton icon={Trash2} idleLabel="Revoke" onConfirm={onConfirm} />);

    await user.click(screen.getByRole("button", { name: "Revoke" }));
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(onConfirm).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Revoke" })).toBeInTheDocument();
  });

  it("auto-reverts to idle after the arm window elapses", () => {
    vi.useFakeTimers();
    const onConfirm = vi.fn();
    render(<ConfirmDestroyButton icon={Trash2} idleLabel="Revoke" onConfirm={onConfirm} armMs={100} />);

    fireEvent.click(screen.getByRole("button", { name: "Revoke" }));
    expect(screen.getByRole("button", { name: "Confirm" })).toBeInTheDocument();

    act(() => vi.advanceTimersByTime(150));
    expect(screen.getByRole("button", { name: "Revoke" })).toBeInTheDocument();
  });

  it("does not arm when disabled", async () => {
    const onConfirm = vi.fn();
    const user = userEvent.setup();
    render(<ConfirmDestroyButton icon={Trash2} idleLabel="Revoke" onConfirm={onConfirm} disabled />);

    await user.click(screen.getByRole("button", { name: "Revoke" }));

    expect(screen.queryByRole("button", { name: "Confirm" })).not.toBeInTheDocument();
  });
});
