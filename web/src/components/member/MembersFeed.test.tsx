import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { MembersFeed } from "@/components/member/MembersFeed";

describe("MembersFeed", () => {
  it("renders the current workspace roster without pending login invites", () => {
    render(<MembersFeed members={[{ user_id: "u-1", login: "alice", role_id: "r-1" }]} roles={[{ id: "r-1", workspace_id: "ws-1", name: "Editor", permissions: [], is_owner_role: false, created_at: "", updated_at: "" }]} isRemoving={false} onRoleChange={vi.fn()} onRemove={vi.fn()} />);
    expect(screen.getByText("alice")).toBeInTheDocument();
    expect(screen.queryByText("Invite sent")).not.toBeInTheDocument();
  });
});
