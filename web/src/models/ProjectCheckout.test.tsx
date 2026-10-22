// @vitest-environment node
import { describe, expect, it } from "vitest";

import type { Computer } from "@/models/Pairing";
import { projectCheckout, projectRemotes, withCheckout } from "@/models/ProjectCheckout";

const computer = (projects?: { id: string; repository?: string }[]) =>
  ({ id: "c-1", facts: projects && { projects: projects.map((p) => ({ title: p.id, path: `/code/${p.id}`, ...p })) } }) as Computer;

describe("projectCheckout", () => {
  const remotes = projectRemotes([
    { owner: "acme", name: "App", full_name: "acme/App", connector_id: "github", role: "app" },
    { owner: "acme", name: "lab", full_name: "acme/lab", connector_id: "gitlab", role: "app" },
  ]);

  it.each([
    ["a computer that never reported", computer(), [], "unknown"],
    ["the one checkout whose remote is the project's", computer([{ id: "t3-app", repository: "github.com/acme/app" }]), [], "has"],
    ["two checkouts of the project", computer([{ id: "a", repository: "github.com/acme/app" }, { id: "b", repository: "github.com/acme/app" }]), [], "ambiguous"],
    ["no checkout of it, and a remote on another host", computer([{ id: "lab", repository: "gitlab.com/acme/lab" }]), [], "missing"],
    ["a picked checkout over the remote", computer([{ id: "a", repository: "github.com/acme/app" }, { id: "b" }]), [{ computer_id: "c-1", harness_project_id: "b" }], "has"],
  ] as const)("%s", (_, c, checkouts, state) => {
    expect(projectCheckout(c, [...checkouts], remotes).state).toBe(state);
  });

  it("falls back to the remote when the picked checkout is gone", () => {
    const got = projectCheckout(computer([{ id: "a", repository: "github.com/acme/app" }]), [{ computer_id: "c-1", harness_project_id: "gone" }], remotes);
    expect(got).toMatchObject({ state: "has", picked: false, project: { id: "a" } });
  });

  it("replaces the checkout picked on a computer", () => {
    const before = [{ computer_id: "c-1", harness_project_id: "a" }, { computer_id: "c-2", harness_project_id: "x" }];
    expect(withCheckout(before, "c-1", "b")).toEqual([{ computer_id: "c-2", harness_project_id: "x" }, { computer_id: "c-1", harness_project_id: "b" }]);
  });
});
