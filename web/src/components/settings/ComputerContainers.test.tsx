import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ComputerDetails } from "@/components/settings/ComputerDetails";
import type { ComputerFacts } from "@/models/ComputerFacts";
import type { Computer } from "@/models/Pairing";

const mocks = vi.hoisted(() => ({ get: vi.fn(), useContainerLogs: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: (e: Error) => e.message }));
vi.mock("@/components/settings/ComputerMCPToken", () => ({ ComputerMCPToken: () => null }));
vi.mock("@/hooks/ContainerLogHooks", () => ({ useContainerLogs: mocks.useContainerLogs }));

const laptop = (docker: ComputerFacts["docker"]): Computer =>
  ({
    id: "c-laptop",
    name: "Laptop",
    kind: "t3code",
    server_url: "http://c-laptop.nexul-computer.invalid",
    token_expires_at: "2099-01-01T00:00:00Z",
    harness_version: "0.0.46",
    created_at: "2026-10-01T00:00:00Z",
    position: 0,
    runs_at_once: 1,
    running: 0,
    chat_project_id: "",
    skills_outdated: false,
    updated_at: "2026-10-01T00:00:00Z",
    facts: { t3: { state: "answering" }, providers: [], projects: [], ...(docker && { docker }) },
  }) as Computer;

const renderDetails = (computer: Computer) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ComputerDetails computer={computer} setup={undefined} />
    </QueryClientProvider>,
  );

describe("ComputerContainers", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.useContainerLogs.mockReset();
    mocks.useContainerLogs.mockReturnValue({
      lines: [{ seq: 0, ts: "2026-10-11T10:00:00Z", stream: "stdout", text: "listening on :8080" }],
      status: "live",
      reason: undefined,
    });
    mocks.get.mockResolvedValue({ data: { containers: [{ name: "shop-web-1", image: "shop:1", status: "running" }] } });
  });

  it("lists the computer's containers and opens one's logs in the stack page's viewer", async () => {
    const user = userEvent.setup();
    renderDetails(laptop({ available: true, version: "27.3.1" }));

    expect(screen.getByText("Docker 27.3.1")).toBeInTheDocument();
    expect(await screen.findByText("shop-web-1")).toBeInTheDocument();
    expect(screen.getByText("running")).toBeInTheDocument();
    expect(mocks.get).toHaveBeenCalledWith("/api/pairing/computers/c-laptop/containers");

    await user.click(screen.getByRole("button", { name: "Logs of shop-web-1" }));
    expect(await screen.findByText("listening on :8080")).toBeInTheDocument();
    expect(mocks.useContainerLogs).toHaveBeenCalledWith("/pairing/computers/c-laptop/containers/shop-web-1/logs");
  });

  it("says so when Docker runs nothing", async () => {
    mocks.get.mockResolvedValue({ data: { containers: [] } });
    renderDetails(laptop({ available: true }));
    expect(await screen.findByText("No containers on this computer.")).toBeInTheDocument();
  });

  it.each([
    ["Docker not reachable", { available: false }],
    ["a runner too old to report Docker", undefined],
  ])("shows no Containers section and asks nothing for %s", (_, docker) => {
    renderDetails(laptop(docker));
    expect(screen.queryByText("Containers")).not.toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalled();
  });
});
