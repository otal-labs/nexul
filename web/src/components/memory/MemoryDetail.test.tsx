import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { MemoryDetail } from "@/components/memory/MemoryDetail";
import type { Memory } from "@/models/Memory";

vi.mock("@/components/attachment/AttachmentsSection", () => ({
  AttachmentsSection: () => <div data-testid="attachments-section" />,
}));
vi.mock("@/components/memory/MemoryVersionsFeed", () => ({
  MemoryVersionsFeed: () => <div data-testid="versions-feed" />,
}));
vi.mock("@/components/memory/CloneMemoryDialog", () => ({
  CloneMemoryDialog: ({ open }: { open: boolean }) => (open ? <div data-testid="clone-dialog" /> : null),
}));
vi.mock("@/components/doc/RichTextEditor", () => ({
  RichTextEditor: () => <div data-testid="rich-text-editor" />,
}));

const memory: Memory = {
  id: "mem-1",
  workspace_id: "ws-1",
  project_id: "project-1",
  kind: "",
  title: "Deploy quirks",
  when_to_use: "when deploying",
  body: "body",
  always_included: false,
  footer: false,
  version: 3,
  created_by: "user-1",
  created_at: "2026-09-16T12:00:00Z",
  updated_by: "user-1",
  updated_at: "2026-09-16T12:00:00Z",
};

const renderDetail = (overrides: Partial<React.ComponentProps<typeof MemoryDetail>> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/memories/m1"]}>
        <Routes>
          <Route
            path="/memories/:memoryId/:tab?"
            element={
              <MemoryDetail
                memory={memory}
                canWrite={false}
                canDelete={false}
                canClone={false}
                onSave={vi.fn()}
                onDelete={vi.fn()}
                saving={false}
                {...overrides}
              />
            }
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("MemoryDetail", () => {
  it("opens the tabs on Attachments and switches to version history for every reader", async () => {
    const user = userEvent.setup();
    renderDetail();
    const panel = screen.getByRole("tabpanel");
    expect(within(panel).getByTestId("attachments-section")).toBeInTheDocument();
    expect(within(panel).queryByTestId("versions-feed")).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Versions" }));
    expect(within(screen.getByRole("tabpanel")).getByTestId("versions-feed")).toBeInTheDocument();
    expect(within(screen.getByRole("tabpanel")).queryByTestId("attachments-section")).not.toBeInTheDocument();
  });

  it("keeps both sections in the rail whichever tab is open", async () => {
    const user = userEvent.setup();
    renderDetail();
    expect(screen.getAllByTestId("attachments-section")).toHaveLength(2);
    expect(screen.getAllByTestId("versions-feed")).toHaveLength(1);

    await user.click(screen.getByRole("tab", { name: "Versions" }));
    expect(screen.getAllByTestId("attachments-section")).toHaveLength(1);
    expect(screen.getAllByTestId("versions-feed")).toHaveLength(2);
  });

  it("hides Clone to… without memories:clone", () => {
    renderDetail({ canClone: false });
    expect(screen.queryByRole("button", { name: "Clone to…" })).not.toBeInTheDocument();
  });

  it("shows Clone to… and opens the dialog with memories:clone", async () => {
    renderDetail({ canClone: true });
    const button = screen.getByRole("button", { name: "Clone to…" });
    expect(screen.queryByTestId("clone-dialog")).not.toBeInTheDocument();
    button.click();
    expect(await screen.findByTestId("clone-dialog")).toBeInTheDocument();
  });

  it("hides Delete without memories:delete", () => {
    renderDetail({ canDelete: false });
    expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
  });

  it("shows the read-only title and when-to-use without memories:write", () => {
    renderDetail({ canWrite: false });
    expect(screen.getByRole("heading", { name: "Deploy quirks" })).toBeInTheDocument();
    expect(screen.getByText("when deploying")).toBeInTheDocument();
    expect(screen.queryByLabelText("Title")).not.toBeInTheDocument();
  });

  it("shows editable fields with memories:write", () => {
    renderDetail({ canWrite: true });
    expect(screen.getByLabelText("Title")).toBeInTheDocument();
    expect(screen.getByTestId("rich-text-editor")).toBeInTheDocument();
  });

  it("locks the interview memory on and counts it against the cap", () => {
    renderDetail({ canWrite: true, memory: { ...memory, kind: "interview", always_included: true, body: "x".repeat(8001) } });
    expect(screen.queryByLabelText("Always included")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Footer")).not.toBeInTheDocument();
    expect(screen.getByText(/can't be switched off/)).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("8,001 / 8,000 characters");
  });

  it("keeps the decisions log out of every turn, with no switch to turn it on", () => {
    renderDetail({ canWrite: true, memory: { ...memory, kind: "decisions_log" } });
    expect(screen.queryByLabelText("Always included")).not.toBeInTheDocument();
    expect(screen.getByText(/never sent in every turn/)).toBeInTheDocument();
  });

  it("shows no cap for an ordinary memory", () => {
    renderDetail({ canWrite: true });
    expect(screen.getByLabelText("Always included")).toBeInTheDocument();
    expect(screen.getByLabelText("Footer")).toBeInTheDocument();
    expect(screen.queryByText(/characters/)).not.toBeInTheDocument();
  });
});
