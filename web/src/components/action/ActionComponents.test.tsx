import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RocketIcon } from "lucide-react";
import type { ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ActionState } from "@nexul/client-core/actionState";

import { ActionButton } from "@/components/action/ActionButton";
import { ActionDragHandle } from "@/components/action/ActionDragHandle";
import { ActionInlineEdit } from "@/components/action/ActionInlineEdit";
import { ActionMenuItem } from "@/components/action/ActionMenuItem";

const mocks = vi.hoisted(() => ({ useActionState: vi.fn() }));

vi.mock("@/hooks/useActionState", () => ({ useActionState: mocks.useActionState }));

const REASON = "Your role can't delete roles";

const controls: { name: string; control: string; tooltip: boolean; draw: (activate: () => void) => ReactElement }[] = [
  {
    name: "button",
    control: "Deploy",
    tooltip: true,
    draw: (activate) => (
      <ActionButton permission="deploys:write" icon={RocketIcon} onClick={activate}>
        Deploy
      </ActionButton>
    ),
  },
  {
    name: "icon button",
    control: "Deploy",
    tooltip: true,
    draw: (activate) => <ActionButton permission="deploys:write" icon={RocketIcon} size="icon" aria-label="Deploy" onClick={activate} />,
  },
  {
    name: "menu item",
    control: "Delete",
    tooltip: false,
    draw: (activate) => (
      <ActionMenuItem permission="roles:delete" destructive onSelect={activate}>
        Delete
      </ActionMenuItem>
    ),
  },
  {
    name: "inline edit",
    control: "Rename Atlas",
    tooltip: true,
    draw: (activate) => (
      <ActionInlineEdit permission="projects:write" label="Rename Atlas" onEdit={activate}>
        Atlas
      </ActionInlineEdit>
    ),
  },
  {
    name: "drag handle",
    control: "Reorder Atlas",
    tooltip: true,
    draw: (activate) => <ActionDragHandle permission="projects:write" label="Reorder Atlas" onClick={activate} />,
  },
];

const locked: ActionState = { state: "locked", reason: REASON };

describe.each(controls)("$name", ({ control, tooltip, draw }) => {
  beforeEach(() => mocks.useActionState.mockReset());

  it("runs its action while live", async () => {
    mocks.useActionState.mockReturnValue({ state: "live" });
    const activate = vi.fn();
    render(draw(activate));

    await userEvent.setup().click(screen.getByRole("button", { name: control }));

    expect(activate).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: control })).not.toHaveAttribute("aria-disabled");
  });

  it("is absent while hidden", () => {
    mocks.useActionState.mockReturnValue({ state: "hidden" });
    render(draw(vi.fn()));

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("stays in the tab order while locked, announces its reason, and does nothing when activated", async () => {
    mocks.useActionState.mockReturnValue(locked);
    const activate = vi.fn();
    const user = userEvent.setup();
    render(draw(activate));

    await user.tab();
    const button = screen.getByRole("button", { name: control });
    expect(button).toHaveFocus();
    expect(button).toHaveAttribute("aria-disabled", "true");
    expect(button).not.toBeDisabled();
    expect(button).toHaveAccessibleDescription(REASON);

    await user.keyboard("{Enter}");
    await user.keyboard(" ");
    await user.click(button);
    expect(activate).not.toHaveBeenCalled();
  });

  it("shows the reason in a tooltip when it takes keyboard focus", async () => {
    mocks.useActionState.mockReturnValue(locked);
    render(draw(vi.fn()));
    await userEvent.setup().tab();

    if (tooltip) expect(await screen.findByRole("tooltip")).toHaveTextContent(REASON);
    if (!tooltip) expect(screen.getByText(REASON)).toBeVisible();
  });
});
