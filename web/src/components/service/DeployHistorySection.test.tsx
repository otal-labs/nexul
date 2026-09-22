import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DeployHistorySection } from "@/components/service/DeployHistorySection";
import { DeployStatus, DeployStrategy, type Deploy } from "@/models/Stack";

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

  it("shows a no-data state instead of a silent empty list", () => {
    renderSection(<DeployHistorySection deploys={[]} isLoading={false} />);
    expect(screen.getByText("No deploys yet.")).toBeInTheDocument();
  });

  it("shows no-data even when deploys are undefined", () => {
    renderSection(<DeployHistorySection deploys={undefined} isLoading={false} />);
    expect(screen.getByText("No deploys yet.")).toBeInTheDocument();
  });

  it("lists deploys with image, status, mono id, and relative time", () => {
    renderSection(<DeployHistorySection deploys={deploys} isLoading={false} />);
    expect(screen.getByText("ghcr.io/onik/api:v1")).toBeInTheDocument();
    expect(screen.getByText("repo build")).toBeInTheDocument();
    expect(screen.getByText("d-1")).toBeInTheDocument();
    expect(screen.getByText("d-2")).toBeInTheDocument();
    expect(screen.getByText("healthy")).toBeInTheDocument();
    expect(screen.getByText("failed")).toBeInTheDocument();
    expect(screen.getByText("3h ago")).toBeInTheDocument();
    expect(screen.getByText("2h ago")).toBeInTheDocument();
  });

  it("groups same-day deploys under one visible chronological heading", () => {
    renderSection(<DeployHistorySection deploys={deploys} isLoading={false} />);
    expect(screen.getByRole("heading", { name: "Today" })).toBeInTheDocument();
  });

  it("splits deploys from different days into separate headings", () => {
    renderSection(
      <DeployHistorySection
        deploys={[...deploys, { ...deploys[0]!, id: "d-3", created_at: "2026-08-10T09:00:00Z" }]}
        isLoading={false}
      />,
    );
    expect(screen.getByRole("heading", { name: "Today" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Aug 10" })).toBeInTheDocument();
  });
});
