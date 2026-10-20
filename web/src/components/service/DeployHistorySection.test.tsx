import { render, screen } from "@testing-library/react";
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
    renderSection(<DeployHistorySection deploys={[]} isLoading />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it.each([[[]], [undefined]])("shows a no-data state instead of a silent empty list for %j", (list) => {
    renderSection(<DeployHistorySection deploys={list} isLoading={false} />);
    expect(screen.getByText("No deploys yet. Deploy the stack and each run shows here.")).toBeInTheDocument();
  });

  it("shows a failed load as an error, not as a stack that never deployed", () => {
    renderSection(<DeployHistorySection deploys={undefined} isLoading={false} error={new Error("boom")} />);
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(screen.queryByText(/No deploys yet/)).not.toBeInTheDocument();
  });

  it("lists deploys with image, status, and relative time, never the id", () => {
    renderSection(<DeployHistorySection deploys={deploys} isLoading={false} />);
    expect(screen.getByText("ghcr.io/onik/api:v1")).toBeInTheDocument();
    expect(screen.getByText("repo build")).toBeInTheDocument();
    expect(screen.queryByText("d-1")).not.toBeInTheDocument();
    expect(screen.queryByText("d-2")).not.toBeInTheDocument();
    expect(screen.getByText("healthy")).toBeInTheDocument();
    expect(screen.getByText("failed")).toBeInTheDocument();
    expect(screen.getByText("3h ago")).toBeInTheDocument();
    expect(screen.getByText("2h ago")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Today" })).toBeInTheDocument();
  });
});
