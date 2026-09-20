import type { ReactElement } from "react";
import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { CreateServiceForm, emptyServiceForm } from "@/components/service/CreateServiceForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { ServiceFormSchema, type ServiceFormData } from "@/models/Service";
import { pickOption } from "@/test/pickOption";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ServiceHarness = ({ projectId }: { projectId: string }) => {
  const { open } = useFormDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button
        type="button"
        onClick={async () => {
          const result = await open<ServiceFormData>({
            title: "New service",
            schema: ServiceFormSchema,
            okLabel: "Create & deploy",
            form: <CreateServiceForm projectId={projectId} />,
            formOptions: { defaultValues: emptyServiceForm() },
          });
          setResult(result.success ? String((result.data as ServiceFormData & { id?: string })?.id) : "cancelled");
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

const runners = [
  { id: "r1", name: "instance", connected: true, last_seen: "2026-09-03T00:00:00Z", running_job: null, version: "0.1.0" },
];

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(api.post).mockResolvedValue({ data: {} });
  vi.mocked(api.get).mockImplementation((url: string) => {
    if (url === "/api/runners") return Promise.resolve({ data: runners });
    return Promise.resolve({ data: [] });
  });
});

describe("CreateServiceForm", () => {
  it("switches the strategy field between compose and run", async () => {
    const user = userEvent.setup();
    renderWithRoot(<ServiceHarness projectId="proj-1" />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(screen.getByLabelText("Compose directory")).toBeInTheDocument();
    await pickOption(user, "Strategy", "Docker run");
    expect(screen.queryByLabelText("Compose directory")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Docker network")).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("creates a service, triggers its first deploy, and resolves with the id", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockImplementation((url: string) => {
      if (url === "/api/services") return Promise.resolve({ data: { id: "svc-1", name: "api" } });
      if (url === "/api/deploys") return Promise.resolve({ data: {} });
      return Promise.resolve({ data: {} });
    });
    renderWithRoot(<ServiceHarness projectId="proj-1" />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByLabelText("Name"), "api");
    await pickOption(user, "Runs on", "instance");
    await user.type(screen.getByLabelText("Health check URL"), "http://10.0.0.1:8080/health");
    await user.type(screen.getByLabelText("Compose directory"), "/srv/api");
    await user.type(screen.getByLabelText("Image (pre-built)"), "ghcr.io/onik/api:v1");
    await user.click(screen.getByRole("button", { name: /create & deploy/i }));

    expect(await screen.findByText("svc-1")).toBeInTheDocument();
    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/api/services", expect.objectContaining({ name: "api" })),
    );
    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/api/deploys", {
        service_id: "svc-1",
        image: "ghcr.io/onik/api:v1",
      }),
    );
  });

  it("creates a repo-driven service and triggers a ref deploy", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockImplementation((url: string) => {
      if (url === "/api/services") return Promise.resolve({ data: { id: "svc-1", name: "api" } });
      if (url === "/api/deploys") return Promise.resolve({ data: {} });
      return Promise.resolve({ data: {} });
    });
    renderWithRoot(<ServiceHarness projectId="proj-1" />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await pickOption(user, "Strategy", "Docker run");
    await user.type(screen.getByLabelText("Name"), "api");
    await pickOption(user, "Runs on", "instance");
    await user.type(screen.getByLabelText("Health check URL"), "http://10.0.0.1:8080/health");
    await user.type(screen.getByLabelText("Docker network"), "app-net");
    await user.type(screen.getByLabelText("Repo owner"), "onik97");
    await user.type(screen.getByLabelText("Repo name"), "api");
    await user.type(screen.getByLabelText("Default branch"), "main");
    await user.type(screen.getByLabelText("Ref to build (repo-driven)"), "main");
    await user.click(screen.getByRole("button", { name: /create & deploy/i }));

    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith(
        "/api/services",
        expect.objectContaining({
          name: "api",
          build_source: expect.objectContaining({ repo_owner: "onik97", repo_name: "api", branch: "main" }),
        }),
      ),
    );
    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/api/deploys", {
        service_id: "svc-1",
        ref: "main",
      }),
    );
  });

  it("surfaces a root error when no project is chosen", async () => {
    const user = userEvent.setup();
    renderWithRoot(<ServiceHarness projectId="" />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByLabelText("Name"), "api");
    await pickOption(user, "Runs on", "instance");
    await user.type(screen.getByLabelText("Health check URL"), "http://10.0.0.1:8080/health");
    await user.type(screen.getByLabelText("Compose directory"), "/srv/api");
    await user.click(screen.getByRole("button", { name: /create & deploy/i }));

    expect(await screen.findByText("Choose a project first")).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
    await user.keyboard("{Escape}");
  });
});
