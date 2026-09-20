import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { MembersFeed } from "@/components/member/MembersFeed";
import type { MemberView, WorkspaceInvite } from "@/models/Member";
import type { Role } from "@/models/Role";

const roles: Role[] = [
  { id: "role-owner", workspace_id: "ws-1", name: "Owner", permissions: [], is_owner_role: true, created_at: "", updated_at: "" },
  { id: "role-editor", workspace_id: "ws-1", name: "Editor", permissions: [], is_owner_role: false, created_at: "", updated_at: "" },
];

const alice: MemberView = { user_id: "u-alice", login: "alice", role_id: "role-owner" };
const bob: MemberView = { user_id: "u-bob", login: "bob", role_id: "role-editor" };
const carolInvite: WorkspaceInvite = {
  workspace_id: "ws-1",
  login: "carol",
  role_id: "role-editor",
  invited_by: "u-alice",
  created_at: "",
};

const noop = () => {};

describe("MembersFeed", () => {
  it("renders a member row per member: avatar, name, mono handle, role select", () => {
    render(
      <MembersFeed
        members={[alice, bob]}
        invites={[]}
        roles={roles}
        isRemoving={false}
        isCancelling={false}
        onRoleChange={noop}
        onRemove={noop}
        onCancelInvite={noop}
      />,
    );

    expect(screen.getByText("alice")).toBeInTheDocument();
    expect(screen.getByText("@alice")).toBeInTheDocument();
    expect(screen.getByText("bob")).toBeInTheDocument();

    const avatars = document.querySelectorAll("img");
    expect(avatars[0]).toHaveAttribute("src", "https://github.com/alice.png");
    expect(avatars[1]).toHaveAttribute("src", "https://github.com/bob.png");
  });

  it("shows the Owner badge (no role select, no remove) for the owner-role member", () => {
    render(
      <MembersFeed
        members={[alice, bob]}
        invites={[]}
        roles={roles}
        isRemoving={false}
        isCancelling={false}
        onRoleChange={noop}
        onRemove={noop}
        onCancelInvite={noop}
      />,
    );

    expect(screen.getByText("Owner")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove alice" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove bob" })).toBeInTheDocument();
  });

  it("renders pending invites with an Invite sent chip and a cancel control", () => {
    render(
      <MembersFeed
        members={[]}
        invites={[carolInvite]}
        roles={roles}
        isRemoving={false}
        isCancelling={false}
        onRoleChange={noop}
        onRemove={noop}
        onCancelInvite={noop}
      />,
    );

    expect(screen.getByText("carol")).toBeInTheDocument();
    expect(screen.getByText("Invite sent")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Withdraw invite for carol" })).toBeInTheDocument();
  });

  it("removes a member via the roster's remove button", async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    render(
      <MembersFeed
        members={[bob]}
        invites={[]}
        roles={roles}
        isRemoving={false}
        isCancelling={false}
        onRoleChange={noop}
        onRemove={onRemove}
        onCancelInvite={noop}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Remove bob" }));
    expect(onRemove).toHaveBeenCalledWith("u-bob");
  });

  it("cancels a pending invite via its withdraw button", async () => {
    const user = userEvent.setup();
    const onCancelInvite = vi.fn();
    render(
      <MembersFeed
        members={[]}
        invites={[carolInvite]}
        roles={roles}
        isRemoving={false}
        isCancelling={false}
        onRoleChange={noop}
        onRemove={noop}
        onCancelInvite={onCancelInvite}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Withdraw invite for carol" }));
    expect(onCancelInvite).toHaveBeenCalledWith("carol");
  });

  it("disables the remove button while a removal is in flight", () => {
    render(
      <MembersFeed
        members={[bob]}
        invites={[]}
        roles={roles}
        isRemoving
        isCancelling={false}
        onRoleChange={noop}
        onRemove={noop}
        onCancelInvite={noop}
      />,
    );

    expect(screen.getByRole("button", { name: "Remove bob" })).toBeDisabled();
  });

  it("falls back to a letter avatar when the GitHub image fails", () => {
    render(
      <MembersFeed
        members={[bob]}
        invites={[]}
        roles={roles}
        isRemoving={false}
        isCancelling={false}
        onRoleChange={noop}
        onRemove={noop}
        onCancelInvite={noop}
      />,
    );

    fireEvent.error(document.querySelector("img")!);
    expect(screen.getByText("B")).toBeInTheDocument();
  });
});
