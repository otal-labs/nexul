import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DeployHistorySection } from "@/components/service/DeployHistorySection";
import { DeployStatus, DeployStrategy, type Deploy } from "@/models/Stack";

const access = vi.hoisted(() => ({ areas: ["deploys"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area) }));

const deploys: Deploy[] = [
  {
    id: "d-1",
    stack_id: "stack-1",
    service: "api",
    target: "10.0.0.1:22",
    image: "ghcr.io/onik/api:v1",
    status: DeployStatus.Healthy,
    strategy: DeployStrategy.Compose,
    created_at: "2026-08-12T09:00:00Z",
    updated_at: "2026-08-12T09:00:00Z",
  },
  {
    id: "d-2",
    stack_id: "stack-1",
    service: "api",
    target: "10.0.0.1:22",
    image: "",
    status: DeployStatus.Failed,
    strategy: DeployStrategy.Compose,
    created_at: "2026-08-12T10:00:00Z",
    updated_at: "2026-08-12T10:00:00Z",
  },
];

const renderSection = (ui: React.ReactElement) => render(<MemoryRouter>{ui}</MemoryRouter>);

describe("DeployHistorySection", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-12T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows loading first", () => {
    renderSection(<DeployHistorySection deploys={[]} lastHealthy={undefined} isLoading />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it.each([[[]], [undefined]])("shows a no-data state instead of a silent empty list for %j", (list) => {
    renderSection(<DeployHistorySection deploys={list} lastHealthy={undefined} isLoading={false} />);
    expect(screen.getByText("No deploys yet. Deploy the stack and each run shows here.")).toBeInTheDocument();
  });

  it("shows a failed load as an error, not as a stack that never deployed", () => {
    renderSection(<DeployHistorySection deploys={undefined} lastHealthy={undefined} isLoading={false} error={new Error("boom")} />);
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(screen.queryByText(/No deploys yet/)).not.toBeInTheDocument();
  });

  it("lists each deploy on one row by its image's last segment, its outcome and when, never the id", () => {
    renderSection(<DeployHistorySection deploys={deploys} lastHealthy={deploys[0]} isLoading={false} />);
    expect(screen.getByTitle("ghcr.io/onik/api:v1")).toHaveTextContent("api:v1");
    expect(screen.getByText("repository")).toBeInTheDocument();
    expect(screen.queryByText("d-1")).not.toBeInTheDocument();
    const list = within(screen.getByRole("list"));
    expect(list.getByText("Succeeded")).toBeInTheDocument();
    expect(list.getByText("Failed")).toBeInTheDocument();
    expect(screen.queryByText("healthy")).not.toBeInTheDocument();
    expect(screen.getAllByText(/^12 Aug \d\d:\d\d$/)).toHaveLength(2);
    expect(screen.getAllByText(/^current/).length).toBeGreaterThan(0);
  });

  it("narrows to the failed deploys and back", async () => {
    vi.useRealTimers();
    const user = userEvent.setup();
    renderSection(<DeployHistorySection deploys={deploys} lastHealthy={deploys[0]} isLoading={false} />);
    await user.click(screen.getByRole("radio", { name: "Failed" }));
    expect(within(screen.getByRole("list")).queryByText("Succeeded")).not.toBeInTheDocument();
    expect(within(screen.getByRole("list")).getByText("Failed")).toBeInTheDocument();
    await user.click(screen.getByRole("radio", { name: "All" }));
    expect(within(screen.getByRole("list")).getByText("Succeeded")).toBeInTheDocument();
  });

  it("shows the newest 25 and the rest behind one button", async () => {
    vi.useRealTimers();
    const user = userEvent.setup();
    const many = Array.from({ length: 30 }, (_, i) => ({ ...deploys[0]!, id: `d-${i}`, image: `ghcr.io/onik/api:v${i}` }));
    renderSection(<DeployHistorySection deploys={many} lastHealthy={undefined} isLoading={false} />);
    expect(screen.getAllByRole("link")).toHaveLength(25);
    await user.click(screen.getByRole("button", { name: "Show 5 older" }));
    expect(screen.getAllByRole("link")).toHaveLength(30);
  });
});
