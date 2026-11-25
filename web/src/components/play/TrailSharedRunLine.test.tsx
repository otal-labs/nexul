import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TrailSharedRunLine } from "@/components/play/TrailSharedRunLine";
import type { Computer } from "@/models/Pairing";
import type { Trail } from "@/models/Trail";

const mocks = vi.hoisted(() => ({ me: "u-lena" }));

vi.mock("@/hooks/AuthHooks", () => ({ useFetchMe: () => ({ data: { user: { id: mocks.me } } }) }));
vi.mock("@/hooks/PeopleHooks", () => ({
  usePersonLookup: () => (id: string) => ({ login: { "u-lena": "lena", "u-sam": "sam" }[id] ?? id, display_name: "" }),
}));

const trail = (starter: string) => ({ starter_id: starter, workspace_id: "ws-1" }) as Trail;
const laptop = { id: "c-1", name: "Laptop" } as Computer;
const shared = { ...laptop, shared: { owner_id: "u-lena", owner_name: "Lena", permissions: ["computers:see", "computers:run_agents"] } } as Computer;

describe("TrailSharedRunLine", () => {
  beforeEach(() => {
    mocks.me = "u-lena";
  });

  it("names the starter and the owner when someone else ran on the viewer's computer", () => {
    render(<TrailSharedRunLine trail={trail("u-sam")} computer={laptop} />);
    expect(screen.getByText("Run by sam using lena's Laptop")).toBeInTheDocument();
  });

  it("names the owner of a computer shared with the viewer", () => {
    mocks.me = "u-sam";
    render(<TrailSharedRunLine trail={trail("u-sam")} computer={shared} />);
    expect(screen.getByText("Run by sam using Lena's Laptop")).toBeInTheDocument();
  });

  it("says nothing on the owner's own run", () => {
    const { container } = render(<TrailSharedRunLine trail={trail("u-lena")} computer={laptop} />);
    expect(container).toBeEmptyDOMElement();
  });
});
