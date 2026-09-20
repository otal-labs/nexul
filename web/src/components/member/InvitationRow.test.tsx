import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { InvitationRow } from "@/components/member/InvitationRow";

describe("InvitationRow", () => {
  it("shows metadata and revokes after confirmation", async () => {
    const onRevoke = vi.fn();
    const user = userEvent.setup();
    render(<InvitationRow invitation={{ id: "inv-12345678", invited_by: "owner-1", created_at: "2026-09-20T00:00:00Z", expires_at: "2026-09-27T00:00:00Z", grants: [{ workspace_id: "ws-1", workspace_name: "Engineering", role_id: "r-1", role_name: "Editor", allow: [], deny: [] }] }} onRevoke={onRevoke} disabled={false} />);
    expect(screen.getByText(/by owner-1 · created/i)).toBeInTheDocument();
    expect(screen.getByText(/Engineering · Editor/i)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Revoke invitation" }));
    await user.click(screen.getByRole("button", { name: "Confirm" }));
    expect(onRevoke).toHaveBeenCalledWith("inv-12345678");
  });
});
