import type { ReactElement } from "react";
import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PermissionsForm, PermissionsFormSchema, type PermissionsFormData } from "@/components/access/PermissionsForm";
import { useFormDialog } from "@/hooks/useFormDialog";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), put: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const catalog = [
  { value: "docs:read", label: "Read docs", domain: "docs", action: "read" },
  { value: "docs:write", label: "Create and update docs", domain: "docs", action: "write" },
  { value: "docs:delete", label: "Delete docs", domain: "docs", action: "delete" },
  { value: "permissions:write", label: "Manage permissions", domain: "permissions", action: "write" },
  { value: "reviews:read", label: "Read reviews", domain: "reviews", action: "read" },
  { value: "plays:run", label: "Run plays", domain: "plays", action: "run" },
];

const mockGet = (users: unknown[]) => (url: string) => {
  if (url === "/api/permissions/users") return Promise.resolve({ data: { users } });
  if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
  return Promise.reject(new Error(`unexpected GET ${url}`));
};

const PermissionsHarness = ({ docIds }: { docIds: string[] }) => {
  const { open } = useFormDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button
        type="button"
        onClick={async () => {
          const result = await open<PermissionsFormData>({
            title: "Permissions",
            schema: PermissionsFormSchema,
            okLabel: "Apply",
            form: <PermissionsForm resourceType="doc" resourceIds={docIds} />,
          });
          setResult(result.success ? "applied" : "cancelled");
        }}
      >
        Open
      </button>
      <p>{result}</p>
    </div>
  );
};

const PlayPermissionsHarness = ({ playIds }: { playIds: string[] }) => {
  const { open } = useFormDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button
        type="button"
        onClick={async () => {
          const result = await open<PermissionsFormData>({
            title: "Exclude users",
            schema: PermissionsFormSchema,
            okLabel: "Apply",
            form: <PermissionsForm resourceType="play" resourceIds={playIds} />,
          });
          setResult(result.success ? "applied" : "cancelled");
        }}
      >
        Open
      </button>
      <p>{result}</p>
    </div>
  );
};

const renderWithRoot = (ui: ReactElement) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      {ui}
    </QueryClientProvider>,
  );

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.put).mockReset();
});

describe("PermissionsForm", () => {
  it("shows users and only the docs + permissions:write actions from the catalog", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(
      mockGet([{ id: "u1", login: "alice", name: "Alice", can_create_workspace: false }]),
    );
    renderWithRoot(<PermissionsHarness docIds={["doc-1"]} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(await screen.findByText("alice")).toBeInTheDocument();
    expect(screen.getByText("Read docs")).toBeInTheDocument();
    expect(screen.getByText("Create and update docs")).toBeInTheDocument();
    expect(screen.getByText("Delete docs")).toBeInTheDocument();
    expect(screen.getByText("Manage permissions")).toBeInTheDocument();
    expect(screen.queryByText("Read reviews")).not.toBeInTheDocument();
    expect(screen.queryByText("Run plays")).not.toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("applies selected users and actions and resolves success", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(
      mockGet([{ id: "u1", login: "alice", name: "Alice", can_create_workspace: false }]),
    );
    vi.mocked(api.put).mockResolvedValue({});
    renderWithRoot(<PermissionsHarness docIds={["doc-1"]} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.click(await screen.findByText("alice"));
    await user.click(screen.getByText("Read docs"));
    await user.click(screen.getByRole("button", { name: "Apply" }));

    expect(await screen.findByText("applied")).toBeInTheDocument();
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith("/api/permissions", {
        doc_ids: ["doc-1"],
        user_ids: ["u1"],
        actions: ["docs:read"],
        grant: true,
      }),
    );
  });

  it("resolves cancelled on cancel", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(mockGet([]));
    renderWithRoot(<PermissionsHarness docIds={["doc-1"]} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(await screen.findByText("cancelled")).toBeInTheDocument();
  });

  it("disables apply until a user and an action are chosen", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(
      mockGet([{ id: "u1", login: "alice", name: "Alice", can_create_workspace: false }]),
    );
    renderWithRoot(<PermissionsHarness docIds={["doc-1"]} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(await screen.findByText("alice")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Apply" })).toBeDisabled();
    await user.click(screen.getByText("alice"));
    await user.click(screen.getByText("Read docs"));
    expect(screen.getByRole("button", { name: "Apply" })).toBeEnabled();
    await user.keyboard("{Escape}");
  });
});

describe("PermissionsForm (play resource type)", () => {
  it("offers only Run plays and defaults to deny", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(
      mockGet([{ id: "u1", login: "alice", name: "Alice", can_create_workspace: false }]),
    );
    renderWithRoot(<PlayPermissionsHarness playIds={["play-1"]} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(await screen.findByText("Run plays")).toBeInTheDocument();
    expect(screen.queryByText("Read docs")).not.toBeInTheDocument();
    expect(screen.getByText("Allow (checked) / Deny (unchecked)")).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("submits a deny with resource_type and resource_ids, no doc_ids", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(
      mockGet([{ id: "u1", login: "alice", name: "Alice", can_create_workspace: false }]),
    );
    vi.mocked(api.put).mockResolvedValue({});
    renderWithRoot(<PlayPermissionsHarness playIds={["play-1"]} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.click(await screen.findByText("alice"));
    await user.click(screen.getByText("Run plays"));
    await user.click(screen.getByRole("button", { name: "Apply" }));

    expect(await screen.findByText("applied")).toBeInTheDocument();
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith("/api/permissions", {
        resource_type: "play",
        resource_ids: ["play-1"],
        user_ids: ["u1"],
        actions: ["plays:run"],
        grant: false,
      }),
    );
  });
});
