import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { AxiosError } from "axios";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardRepositoryStep } from "@/components/wizard/WizardRepositoryStep";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), errorMessage: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: mocks.errorMessage,
}));

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
