import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { InvitationRow } from "@/components/member/InvitationRow";

describe("InvitationRow", () => {
  it("titles the link by its access, names the inviter, and revokes after confirmation", async () => {
    const onRevoke = vi.fn();
    const user = userEvent.setup();
    render(<InvitationRow inviter="Onik Noor" invitation={{ id: "inv-12345678", invited_by: "owner-1", created_at: "2026-09-20T00:00:00Z", expires_at: new Date(Date.now() + 7 * 86_400_000).toISOString(), grants: [{ workspace_id: "ws-1", workspace_name: "Engineering", role_id: "r-1", role_name: "Editor", allow: [], deny: [] }] }} onRevoke={onRevoke} disabled={false} />);
    expect(screen.getByText("Engineering · Editor")).toBeInTheDocument();
    expect(screen.getByText("by Onik Noor · expires in 7 days")).toBeInTheDocument();
    expect(screen.queryByText(/owner-1|inv-1/)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Revoke invitation" }));
    await user.click(screen.getByRole("button", { name: "Confirm" }));
    expect(onRevoke).toHaveBeenCalledWith("inv-12345678");
  });
});
