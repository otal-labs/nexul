import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { InstanceTemplatesSection } from "@/components/templates/InstanceTemplatesSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Template } from "@/models/Template";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), post: vi.fn(), delete: vi.fn() }));

vi.mock("@/api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/api/client")>()),
  api: mocks,
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const template = (over: Partial<Template>): Template => ({
  kind: "interview",
  key: "",
  name: "Interview",
  scope: "instance",
  body: "## Stack",
  default_body: "## Stack",
  edited: false,
  follows: true,
  ...over,
});

const templates: Template[] = [
  template({
    body: "## Ours",
    questions: [{ text: "Ours", hint: "", multi_select: false, options: [] }],
    edited: true,
    updated_by: "u-1",
    updated_at: new Date().toISOString(),
  }),
  template({ kind: "mention_chip", name: "Mention chip", body: "{ticket.Ticket}", default_body: "{ticket.Ticket}" }),
  template({ kind: "play_instructions", key: "fix-with-ai", name: "Fix with AI", body: "Fix it", default_body: "Fix it", follows: false }),
  template({ kind: "ticket_body", key: "bug", name: "bug", body: "## Steps", default_body: "## Steps", follows: false }),
  template({ kind: "agent_prompt", key: "intro", name: "Intro", body: "You are the Agent.", default_body: "You are the Agent.", follows: false }),
  template({ kind: "agent_prompt", key: "footer", name: "Footer", body: "Reply when done.", default_body: "Reply when done.", follows: false }),
];

// The target the clone dialog reads before overwriting; edited decides whether it asks first.
let target: Partial<Template> = { edited: false };

const routeGet = (url: string) => {
  if (url === "/api/templates") return Promise.resolve({ data: templates });
  if (url.startsWith("/api/templates/")) return Promise.resolve({ data: template({ scope: "workspace", ...target }) });
  if (url === "/api/auth/me") return Promise.resolve({ data: { user: { id: "u-1" }, instance_permissions: ["templates:write"] } });
  if (url === "/api/workspaces") {
    return Promise.resolve({ data: [{ id: "ws-1", name: "Alpha", slug: "alpha" }, { id: "ws-2", name: "Beta", slug: "beta" }] });
  }
  if (url.endsWith("/me")) return Promise.resolve({ data: { role_name: "Owner", permissions: ["memories:write", "plays:write"] } });
  if (url.endsWith("/people")) return Promise.resolve({ data: { people: [{ user_id: "u-1", login: "sam", display_name: "Sam" }] } });
  return Promise.reject(new Error(`unexpected GET ${url}`));
};

const renderSection = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <InstanceTemplatesSection />
    </QueryClientProvider>,
  );

const openEditor = async (label: string) => {
  const user = userEvent.setup();
  renderSection();
  await user.click(await screen.findByRole("button", { name: `Edit ${label}` }));
  return user;
};

const pickTarget = async (user: ReturnType<typeof userEvent.setup>, scope: string, name: string) => {
  await user.click(await screen.findByRole("combobox", { name: "Clone to" }));
  await user.click(await screen.findByRole("option", { name: scope }));
  await user.click(await screen.findByRole("combobox", { name: "Workspace" }));
  await user.click(await screen.findByRole("option", { name }));
  await user.click(screen.getByRole("button", { name: "Clone" }));
};

describe("InstanceTemplatesSection", () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset());
    mocks.get.mockImplementation(routeGet);
    target = { edited: false };
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  });

  it("lists each template under its group with who edited it, or Default", async () => {
    renderSection();

    const interview = within(await screen.findByRole("region", { name: "Interview" }));
    expect(await interview.findByText("Edited")).toBeInTheDocument();
    expect(interview.getByText(/by Sam /)).toBeInTheDocument();
    const plays = within(screen.getByRole("region", { name: "Play instructions" }));
    expect(plays.getByText("Fix with AI")).toBeInTheDocument();
    expect(plays.getByText("Default")).toBeInTheDocument();
    expect(within(screen.getByRole("region", { name: "Ticket bodies" })).getByText("bug body")).toBeInTheDocument();
    const prompt = within(screen.getByRole("region", { name: "Agent prompt" }));
    expect(prompt.getByText("Intro")).toBeInTheDocument();
    expect(prompt.getByText("Footer")).toBeInTheDocument();
  });

  it("saves an edited instance template", async () => {
    mocks.put.mockResolvedValue({ data: {} });
    const user = await openEditor("Interview");
    expect(screen.getByText("1 question")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Template (markdown)"), "!");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/templates/interview", { key: "", body: "## Ours!" });
  });

  it("edits an Agent prompt template and offers no clone, since it lives only at the instance", async () => {
    mocks.put.mockResolvedValue({ data: {} });
    const user = await openEditor("Footer");

    await user.type(screen.getByLabelText("Text"), "!");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/templates/agent_prompt", { key: "footer", body: "Reply when done.!" });
    expect(screen.queryByRole("button", { name: "Clone to…" })).not.toBeInTheDocument();
  });

  it("resets an edited instance template to the default only after a confirm", async () => {
    mocks.delete.mockResolvedValue({ data: {} });
    const user = await openEditor("Interview");

    await user.click(screen.getByRole("button", { name: "Reset to default" }));
    expect(mocks.delete).not.toHaveBeenCalled();
    await user.click(await screen.findByRole("button", { name: "Reset" }));

    expect(mocks.delete).toHaveBeenCalledWith("/api/templates/interview", { params: { key: "" } });
  });

  it("clones an instance template over a workspace's", async () => {
    mocks.post.mockResolvedValue({ data: {} });
    const user = await openEditor("Fix with AI instructions");

    await user.click(screen.getByRole("button", { name: "Clone to…" }));
    await pickTarget(user, "A workspace", "Beta");

    await vi.waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/templates/clone", {
        kind: "play_instructions",
        key: "fix-with-ai",
        from: { scope: "instance" },
        to: { scope: "workspace", workspace_id: "ws-2" },
      }),
    );
  });

  it("shows the server's no-match message inside the dialog", async () => {
    mocks.post.mockRejectedValue({
      response: { status: 404, data: { code: "NOT_FOUND", message: 'workspace has no play with key "fix-with-ai"' } },
    });
    const user = await openEditor("Fix with AI instructions");

    await user.click(screen.getByRole("button", { name: "Clone to…" }));
    await pickTarget(user, "A workspace", "Beta");

    expect(await within(screen.getByRole("dialog")).findByRole("alert")).toHaveTextContent('workspace has no play with key "fix-with-ai"');
    // react-confirm keeps an open dialog across renders, so close it for the next test.
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });

  it("asks before overwriting a target with its own edits, and clones nothing when told no", async () => {
    target = { edited: true };
    const user = await openEditor("Interview");

    await user.click(screen.getByRole("button", { name: "Clone to…" }));
    await pickTarget(user, "A workspace", "Beta");
    expect(await screen.findByText("That workspace has its own text for this template. Cloning overwrites it.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(mocks.post).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Clone" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });
});
