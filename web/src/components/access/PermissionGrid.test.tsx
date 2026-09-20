import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { PermissionGrid } from "@/components/access/PermissionGrid";
import type { PermissionInfo } from "@/models/Permission";

const catalog: PermissionInfo[] = [
  { value: "docs:read", label: "Read docs", domain: "docs", action: "read" },
  { value: "docs:write", label: "Create and update docs", domain: "docs", action: "write" },
  { value: "docs:thread", label: "See doc threads", domain: "docs", action: "thread" },
  { value: "reviews:read", label: "Read reviews", domain: "reviews", action: "read" },
  { value: "deploys:read", label: "Read deploys", domain: "deploys", action: "read" },
  { value: "deploys:write", label: "Trigger deploys", domain: "deploys", action: "write" },
  { value: "plays:read", label: "Read plays", domain: "plays", action: "read" },
  { value: "plays:run", label: "Run plays", domain: "plays", action: "run" },
];

describe("PermissionGrid", () => {
  it("renders one row per domain and only the actions the catalog actually offers", () => {
    render(<PermissionGrid entries={catalog} value={[]} onChange={vi.fn()} />);

    expect(screen.getByText("Docs")).toBeInTheDocument();
    expect(screen.getByText("Reviews")).toBeInTheDocument();
    expect(screen.getByText("Deploys")).toBeInTheDocument();
    expect(screen.getByText("Plays")).toBeInTheDocument();

    // Columns are derived from the catalog in first-appearance order: read, write, thread, run.
    expect(screen.getByText("Read")).toBeInTheDocument();
    expect(screen.getByText("Write")).toBeInTheDocument();
    expect(screen.getByText("Thread")).toBeInTheDocument();
    expect(screen.getByText("Run")).toBeInTheDocument();

    expect(screen.getByRole("checkbox", { name: "Read docs" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Create and update docs" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "See doc threads" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Read reviews" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Read deploys" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Trigger deploys" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Read plays" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Run plays" })).toBeInTheDocument();

    // reviews is read-only — no write/thread/run checkboxes leak into that row.
    expect(screen.getAllByRole("checkbox")).toHaveLength(8);
  });

  it("ticking a cell adds its value to the submitted permissions", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<PermissionGrid entries={catalog} value={["docs:read"]} onChange={onChange} />);

    await user.click(screen.getByRole("checkbox", { name: "Trigger deploys" }));

    expect(onChange).toHaveBeenCalledWith(["docs:read", "deploys:write"]);
  });

  it("unticking a cell removes it from the submitted permissions", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<PermissionGrid entries={catalog} value={["docs:read", "docs:write"]} onChange={onChange} />);

    await user.click(screen.getByRole("checkbox", { name: "Create and update docs" }));

    expect(onChange).toHaveBeenCalledWith(["docs:read"]);
  });
});
