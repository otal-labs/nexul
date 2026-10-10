import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { AxiosError } from "axios";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardRepositoryStep } from "@/components/wizard/WizardRepositoryStep";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), errorMessage: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, put: mocks.put },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const repos = [
  { id: 1, owner: "onik97", name: "api", full_name: "onik97/api", default_branch: "main", html_url: "", provider: "github" },
  { id: 2, owner: "onik97", name: "worker", full_name: "onik97/worker", default_branch: "main", html_url: "", provider: "github" },
];

const repositoryRequests = () => mocks.get.mock.calls.filter(([url]) => url === "/api/repositories");

// Types into the deploy search and waits out its debounce for the results.
const search = async (user: ReturnType<typeof userEvent.setup>, text = "onik97") => {
  await user.type(screen.getByLabelText("Search repositories"), text);
  await screen.findByText("onik97/api");
};

const renderStep = (onDone = vi.fn()) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <WizardRepositoryStep onDone={onDone} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return onDone;
};

const notFoundError = (message: string): AxiosError =>
  ({ response: { status: 404, data: { message } } }) as AxiosError;

beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
  mocks.put.mockReset();
  mocks.errorMessage.mockReset();
  mocks.errorMessage.mockImplementation((error: unknown) => (error as { response?: { data?: { message?: string } } })?.response?.data?.message ?? "");
  useProjectWizardStore.getState().reset();
  useWorkspaceStore.getState().selectWorkspace("", "");
  mocks.get.mockResolvedValue({ data: { repositories: repos } });
});

describe("WizardRepositoryStep", () => {
  it("asks for nothing until three letters are typed, then searches the workspace's repositories once for the whole word", async () => {
    useWorkspaceStore.getState().selectWorkspace("ws-1", "acme");
    renderStep();
    const user = userEvent.setup();
    const input = screen.getByLabelText("Search repositories");

    await user.type(input, "on");
    expect(screen.getByText("Type at least 3 letters to search your repositories.")).toBeInTheDocument();
    expect(repositoryRequests()).toHaveLength(0);

    await user.type(input, "ik97");
    expect(await screen.findByText("onik97/api")).toBeInTheDocument();
    expect(repositoryRequests()).toHaveLength(1);
    expect(repositoryRequests()[0]![1]).toEqual(expect.objectContaining({ params: { workspace_id: "ws-1", q: "onik97" } }));
    expect(screen.queryByText(/Type at least 3 letters/)).not.toBeInTheDocument();
  });

  it("marks each row with where it comes from, and nothing for a provider it has no mark for", async () => {
    mocks.get.mockResolvedValue({
      data: { repositories: [repos[0], { ...repos[1]!, provider: "somewhere-else" }] },
    });
    renderStep();
    const user = userEvent.setup();

    await search(user);
    expect(screen.getAllByRole("img", { name: "GitHub" })).toHaveLength(1);
    expect(within(screen.getByRole("button", { name: /onik97\/api/ })).getByRole("img", { name: "GitHub" })).toBeInTheDocument();
  });

  it("says so when nothing matches", async () => {
    mocks.get.mockResolvedValue({ data: { repositories: [] } });
    renderStep();
    const user = userEvent.setup();

    await user.type(screen.getByLabelText("Search repositories"), "zzz");
    expect(await screen.findByText("No repositories match")).toBeInTheDocument();
  });

  it("advances once a scan finds a candidate", async () => {
    useWorkspaceStore.getState().selectWorkspace("ws-1", "acme");
    const onDone = renderStep();
    const user = userEvent.setup();
    mocks.post.mockResolvedValue({
      data: {
        default_branch: "main",
        candidates: [{ kind: "compose", path: "docker-compose.yml", name: "api", services: [] }],
        env_keys: [],
      },
    });

    await search(user);
    await user.click(screen.getByText("onik97/worker"));
    expect(mocks.post).toHaveBeenCalledWith("/api/repositories/scan", { owner: "onik97", name: "worker", workspace_id: "ws-1" });
    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(useProjectWizardStore.getState().candidate?.name).toBe("api");
  });

  it("points at GitHub's install page for the configured App so repositories from other accounts can appear", async () => {
    mocks.get.mockImplementation(async (url: string) => ({
      data: url === "/api/connectors/github/app-config" ? { configured: true, app_slug: "nexul-otal" } : { repositories: repos },
    }));
    renderStep();

    const link = await screen.findByRole("link", { name: "Install it on another account or organisation" });
    expect(link).toHaveAttribute("href", "https://github.com/apps/nexul-otal/installations/new");
    expect(link).toHaveAttribute("target", "_blank");
    expect(screen.getByText(/installs the App there and gives the connected account access to it/)).toBeInTheDocument();
    expect(screen.getByText(/Only the connected account's repositories are visible until the App's private key is added/)).toBeInTheDocument();
  });

  it("once Nexul reads GitHub as the App, links an install that lands in this workspace and drops the connected-account signal", async () => {
    useWorkspaceStore.getState().selectWorkspace("ws-1", "acme");
    const stateURL = "https://github.com/apps/nexul-otal/installations/new?state=install.N0NCE";
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/connectors/github/app-config") return { data: { configured: true, app_slug: "nexul-otal", private_key_set: true } };
      if (url === "/api/repositories/install-url") return { data: { url: stateURL } };
      return { data: { repositories: repos } };
    });
    renderStep();

    expect(await screen.findByText(/lists the repositories of the GitHub accounts assigned to it/)).toBeInTheDocument();
    await vi.waitFor(() =>
      expect(screen.getByRole("link", { name: "Install it on another account or organisation" })).toHaveAttribute("href", stateURL),
    );
    expect(screen.queryByText(/Only the connected account's repositories are visible/)).not.toBeInTheDocument();
    expect(screen.getByText(/adds the account to this workspace when the installer owns it/)).toBeInTheDocument();
  });

  it("as the App but with no install link of this workspace's, says an install from the plain link waits unassigned", async () => {
    useWorkspaceStore.getState().selectWorkspace("ws-1", "acme");
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/connectors/github/app-config") return { data: { configured: true, app_slug: "nexul-otal", private_key_set: true } };
      if (url === "/api/repositories/install-url") throw new Error("forbidden");
      return { data: { repositories: repos } };
    });
    renderStep();

    expect(await screen.findByText(/waits unassigned until someone who manages connectors assigns it/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Install it on another account or organisation" })).toHaveAttribute(
      "href",
      "https://github.com/apps/nexul-otal/installations/new",
    );
    expect(screen.queryByText(/adds the account to this workspace/)).not.toBeInTheDocument();
  });

  it("explains where repositories come from without a broken link when no App slug is configured", async () => {
    mocks.get.mockImplementation(async (url: string) => ({
      data: url === "/api/connectors/github/app-config" ? { configured: false } : { repositories: repos },
    }));
    renderStep();

    expect(await screen.findByText(/Nexul reads GitHub as the account connected in Settings → Connectors/)).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /Install it on another account/ })).not.toBeInTheDocument();
  });

  it("shows an install link when the App isn't installed on the picked repository", async () => {
    renderStep();
    const user = userEvent.setup();
    mocks.post.mockRejectedValue(
      notFoundError("repository not found, or the GitHub App is not installed on it: install it at https://github.com/apps/nexul/installations/new"),
    );

    await search(user);
    await user.click(screen.getByText("onik97/api"));
    expect(await screen.findByText("The GitHub App isn't installed on this repository")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /install the github app/i })).toHaveAttribute(
      "href",
      "https://github.com/apps/nexul/installations/new",
    );
  });

  it("offers a manual candidate when the scan finds nothing to deploy", async () => {
    const onDone = renderStep();
    const user = userEvent.setup();
    mocks.post.mockResolvedValue({ data: { default_branch: "main", candidates: [], env_keys: [] } });

    await search(user);
    await user.click(screen.getByText("onik97/api"));
    expect(await screen.findByText("Nothing to deploy found")).toBeInTheDocument();
    expect(onDone).not.toHaveBeenCalled();

    const pathInput = screen.getByLabelText("Dockerfile path");
    await user.clear(pathInput);
    await user.type(pathInput, "docker/Dockerfile");
    await user.click(screen.getByRole("button", { name: /continue manually/i }));
    expect(onDone).toHaveBeenCalled();
    expect(useProjectWizardStore.getState().candidate).toEqual(
      expect.objectContaining({ kind: "dockerfile", name: "api", path: "docker/Dockerfile" }),
    );
  });
});

describe("WizardRepositoryStep tests question", () => {
  const scanFinds = () =>
    mocks.post.mockImplementation((url: string) =>
      Promise.resolve(
        url === "/api/repositories/scan"
          ? { data: { default_branch: "main", candidates: [{ kind: "compose", path: "docker-compose.yml", name: "api", services: [] }], env_keys: [] } }
          : { data: undefined },
      ),
    );

  beforeEach(() => {
    useProjectWizardStore.getState().setProjectId("p-1", "Backend");
    mocks.get.mockImplementation((url: string) =>
      Promise.resolve(url === "/api/repositories" ? { data: { repositories: repos } } : { data: [] }),
    );
    mocks.put.mockResolvedValue({ data: {} });
  });

  it("records tests in the same repository once the deployed repository is picked", async () => {
    const onDone = renderStep();
    const user = userEvent.setup();
    scanFinds();

    await user.click(await screen.findByRole("radio", { name: /in the repository you deploy/i }));
    await search(user);
    await user.click(screen.getByText("onik97/api"));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.put).toHaveBeenCalledWith("/api/projects/p-1/tests-location", { tests_location: "same" });
  });

  it("attaches a separate tests repository and keeps it out of the deployable list", async () => {
    const onDone = renderStep();
    const user = userEvent.setup();
    scanFinds();

    await user.click(await screen.findByRole("radio", { name: /in a separate repository/i }));
    await user.type(screen.getByLabelText("Search tests repositories"), "onik97");
    await user.click(await screen.findByRole("button", { name: /onik97\/worker/ }));
    await user.clear(screen.getByLabelText("Search tests repositories"));

    await search(user);
    expect(screen.queryByRole("button", { name: /onik97\/worker/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /onik97\/api/ }));
    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.post).toHaveBeenCalledWith("/api/projects/p-1/repos", {
      owner: "onik97",
      name: "worker",
      connector_id: "github",
      role: "tests",
    });
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("stays on the step when the answer cannot be saved", async () => {
    const onDone = renderStep();
    const user = userEvent.setup();
    scanFinds();
    mocks.put.mockRejectedValue(new Error("boom"));

    await user.click(await screen.findByRole("radio", { name: /in the repository you deploy/i }));
    await search(user);
    await user.click(screen.getByText("onik97/api"));

    await vi.waitFor(() => expect(mocks.put).toHaveBeenCalled());
    expect(onDone).not.toHaveBeenCalled();
  });

  it("saves nothing when the question is left unanswered", async () => {
    const onDone = renderStep();
    const user = userEvent.setup();
    scanFinds();

    await search(user);
    await user.click(screen.getByText("onik97/api"));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("is not asked when attaching a repository to an existing stack", async () => {
    useProjectWizardStore.getState().setAttachStackId("s-1");
    renderStep();

    expect(screen.getByLabelText("Search repositories")).toBeInTheDocument();
    expect(screen.queryByText("Where do this project's tests live?")).not.toBeInTheDocument();
  });
});
