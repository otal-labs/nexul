import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TemplateRolesSection } from "@/components/templates/TemplateRolesSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Template } from "@/models/Template";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), post: vi.fn(), delete: vi.fn() }));

vi.mock("@/api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/api/client")>()),
  api: mocks,
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const catalog = [
  { value: "tickets:read", label: "Read tickets", domain: "tickets", action: "read", area: "project" },
  { value: "tickets:write", label: "Create and update tickets", domain: "tickets", action: "write", area: "project" },
  { value: "runners:read", label: "Read runners", domain: "runners", action: "read", area: "instance" },
];

const roleTemplate = (key: string, body: object, over: Partial<Template> = {}): Template => ({
  kind: "role",
  key,
  name: key,
  scope: "instance",
  body: JSON.stringify(body),
  default_body: JSON.stringify(body),
  edited: false,
  follows: false,
  added: false,
  ...over,
});

const templates: Template[] = [
  roleTemplate("client", { name: "Client", description: "Reads the tickets.", permissions: ["tickets:read"], welcome_steps: ["projects"] }),
  roleTemplate("0199-support", { name: "Support", permissions: [], welcome_steps: [] }, { added: true, edited: true, default_body: "" }),
];

const renderSection = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <TemplateRolesSection />
    </QueryClientProvider>,
  );

const menuItems = async (user: ReturnType<typeof userEvent.setup>, name: string) => {
  await user.click(await screen.findByRole("button", { name: `Actions for ${name}` }));
  return within(await screen.findByRole("dialog")).getAllByRole("button").map((b) => b.textContent);
};

describe("TemplateRolesSection", () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset());
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/templates") return Promise.resolve({ data: templates });
      if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
      if (url === "/api/workspaces") return Promise.resolve({ data: [] });
      return Promise.reject(new Error(`unexpected GET ${url}`));
    });
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  });

  it("offers Reset on a shipped template role and Delete on an added one, never both", async () => {
    const user = userEvent.setup();
    renderSection();

    expect(await screen.findByText("Your projects", { exact: false })).toBeInTheDocument();
    expect(await menuItems(user, "Client")).toEqual(["Edit", "Reset to default"]);
    await user.keyboard("{Escape}");
    expect(await menuItems(user, "Support")).toEqual(["Edit", "Delete"]);
  });

  it("creates a template role in the role editor, saved to the instance as its JSON body", async () => {
    mocks.put.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "New template role" }));
    await user.type(screen.getByRole("textbox", { name: "Name" }), "Contractor");
    await user.type(screen.getByRole("textbox", { name: "Description" }), "Works the tickets they're given.");
    await user.click(screen.getByRole("tab", { name: /^Workspace/ }));
    await user.click(screen.getByRole("switch", { name: /^Show locked actions/ }));
    await user.click(screen.getByRole("tab", { name: /^Welcome/ }));
    await user.click(screen.getByRole("switch", { name: /^Your computer/ }));
    await user.click(screen.getByRole("button", { name: "Create role" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/templates/role", { key: "", body: expect.any(String) });
    expect(JSON.parse(mocks.put.mock.calls[0]![1].body)).toEqual({
      name: "Contractor",
      description: "Works the tickets they're given.",
      permissions: [],
      welcome_steps: ["computer"],
      show_locked_actions: true,
    });
  });

  it("edits a template role under its key", async () => {
    mocks.put.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Actions for Client" }));
    await user.click(await screen.findByRole("button", { name: "Edit" }));
    await user.click(screen.getByRole("tab", { name: /^Welcome/ }));
    await user.click(screen.getByRole("switch", { name: /^Connect GitHub/ }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/templates/role", { key: "client", body: expect.any(String) });
    expect(JSON.parse(mocks.put.mock.calls[0]![1].body)).toEqual({
      name: "Client",
      description: "Reads the tickets.",
      permissions: ["tickets:read"],
      welcome_steps: ["github", "projects"],
      show_locked_actions: false,
    });
  });

  it("deletes an added template role only after a confirm", async () => {
    mocks.post.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Actions for Support" }));
    await user.click(await screen.findByRole("button", { name: "Delete" }));
    expect(mocks.post).not.toHaveBeenCalled();
    await user.click(await screen.findByRole("button", { name: "Delete template role" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/templates/delete", { kind: "role", key: "0199-support" });
  });
});
