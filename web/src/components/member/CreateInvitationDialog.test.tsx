import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { CreateInvitationDialog } from "@/components/member/CreateInvitationDialog";

const mocks = vi.hoisted(() => ({ open: vi.fn() }));
vi.mock("@/hooks/useFormDialog", () => ({ useFormDialog: () => ({ open: mocks.open }) }));

describe("CreateInvitationDialog", () => {
  it("reveals the created link once and copies it", async () => {
    mocks.open.mockImplementation(async (options: { form: { props: { onCreated: (value: unknown) => void } } }) => {
      options.form.props.onCreated({ id: "inv-1", invited_by: "owner", created_at: "2026-09-20T00:00:00Z", expires_at: "2026-09-27T00:00:00Z", grants: [], url: "https://nexul.test/invite#secret" });
      return { success: true, data: null };
    });
    const user = userEvent.setup();
    render(<CreateInvitationDialog />);
    await user.click(screen.getByRole("button", { name: "Invite" }));
    expect(await screen.findByText("https://nexul.test/invite#secret")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Copy link" }));
    expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
  });
});
