import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { AxiosError } from "axios";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardRepositoryStep } from "@/components/wizard/WizardRepositoryStep";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), errorMessage: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, put: mocks.put },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const repos = [
  { id: 1, owner: "onik97", name: "api", full_name: "onik97/api", default_branch: "main", html_url: "" },
  { id: 2, owner: "onik97", name: "worker", full_name: "onik97/worker", default_branch: "main", html_url: "" },
];

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
  mocks.get.mockResolvedValue({ data: { repositories: repos } });
});

describe("WizardRepositoryStep", () => {
  it("lists repositories, filters by search, and advances once a scan finds a candidate", async () => {
    const onDone = renderStep();
    const user = userEvent.setup();
    mocks.post.mockResolvedValue({
      data: {
        default_branch: "main",
        candidates: [{ kind: "compose", path: "docker-compose.yml", name: "api", services: [] }],
        env_keys: [],
      },
    });

    expect(await screen.findByText("onik97/api")).toBeInTheDocument();
    expect(screen.getByText("onik97/worker")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Search repositories"), "work");
    expect(screen.queryByText("onik97/api")).not.toBeInTheDocument();
    expect(screen.getByText("onik97/worker")).toBeInTheDocument();

    await user.click(screen.getByText("onik97/worker"));
    expect(mocks.post).toHaveBeenCalledWith("/api/repositories/scan", { owner: "onik97", name: "worker" });
    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(useProjectWizardStore.getState().candidate?.name).toBe("api");
  });

  it("shows an install link when the App isn't installed on the picked repository", async () => {
    renderStep();
    const user = userEvent.setup();
    mocks.post.mockRejectedValue(
      notFoundError("repository not found, or the GitHub App is not installed on it: install it at https://github.com/apps/nexul/installations/new"),
    );

    await user.click(await screen.findByText("onik97/api"));
    expect(await screen.findByText("Not installed on this repository")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /install the github app/i })).toHaveAttribute(
      "href",
      "https://github.com/apps/nexul/installations/new",
    );
  });

  it("offers a manual candidate when the scan finds nothing to deploy", async () => {
    const onDone = renderStep();
    const user = userEvent.setup();
    mocks.post.mockResolvedValue({ data: { default_branch: "main", candidates: [], env_keys: [] } });

    await user.click(await screen.findByText("onik97/api"));
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
    await user.click(await screen.findByText("onik97/api"));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.put).toHaveBeenCalledWith("/api/projects/p-1/tests-location", { tests_location: "same" });
  });

  it("attaches a separate tests repository and keeps it out of the deployable list", async () => {
    const onDone = renderStep();
    const user = userEvent.setup();
    scanFinds();

    await user.click(await screen.findByRole("radio", { name: /in a separate repository/i }));
    await pickOption(user, "Tests repository", "onik97/worker");
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
    await user.click(await screen.findByText("onik97/api"));

    await vi.waitFor(() => expect(mocks.put).toHaveBeenCalled());
    expect(onDone).not.toHaveBeenCalled();
  });

  it("saves nothing when the question is left unanswered", async () => {
    const onDone = renderStep();
    const user = userEvent.setup();
    scanFinds();

    await user.click(await screen.findByText("onik97/api"));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("is not asked when attaching a repository to an existing stack", async () => {
    useProjectWizardStore.getState().setAttachStackId("s-1");
    renderStep();

    expect(await screen.findByText("onik97/api")).toBeInTheDocument();
    expect(screen.queryByText("Where do this project's tests live?")).not.toBeInTheDocument();
  });
});
