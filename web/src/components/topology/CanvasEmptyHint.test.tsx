import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CanvasEmptyHint } from "@/components/topology/CanvasEmptyHint";
import { resetViewer, viewer } from "@/test/viewerActions";

const access = vi.hoisted(() => ({ areas: ["editProjects", "editStacks", "editDeploys"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area) }));

vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);

beforeEach(() => {
  access.areas = ["editProjects", "editStacks", "editDeploys"];
  resetViewer();
  viewer.instance = ["topology:write"];
});

const renderHint = (onAddNode: () => void) =>
  render(
    <MemoryRouter>
      <CanvasEmptyHint onAddNode={onAddNode} />
    </MemoryRouter>,
  );

describe("CanvasEmptyHint", () => {
  it("renders the empty-canvas invitation copy", () => {
    renderHint(() => {});
    expect(screen.getByText("Nothing on the canvas yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /add a node/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /add a service/i })).toHaveAttribute("href", "/acme/wizard/project/repository");
  });

  it("opens the add-node flow from the CTA", () => {
    const onAddNode = vi.fn();
    renderHint(onAddNode);
    fireEvent.click(screen.getByRole("button", { name: /add a node/i }));
    expect(onAddNode).toHaveBeenCalledTimes(1);
  });

  it("hides Add a service without projects:write and keeps Add a node", () => {
    access.areas = [];
    renderHint(() => {});
    expect(screen.queryByRole("link", { name: /add a service/i })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /add a node/i })).toBeInTheDocument();
  });

  it.each([
    ["hides Add a node", false],
    ["locks Add a node", true],
  ])("%s from someone who can't write topology", (_name, showLocked) => {
    viewer.instance = [];
    viewer.showLocked = showLocked;
    const onAddNode = vi.fn();
    renderHint(onAddNode);

    const add = screen.queryByRole("button", { name: /add a node/i });
    expect(!!add).toBe(showLocked);
    if (!add) return;
    fireEvent.click(add);
    expect(add).toHaveAttribute("aria-disabled", "true");
    expect(onAddNode).not.toHaveBeenCalled();
  });
});
