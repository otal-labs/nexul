// @vitest-environment node
import { describe, expect, it } from "vitest";

import type { Computer } from "@/models/Pairing";
import { projectCheckout, projectRemotes, repoCheckouts, withCheckout } from "@/models/ProjectCheckout";

const computer = (projects?: { id: string; repository?: string }[]) =>
  ({ id: "c-1", facts: projects && { projects: projects.map((p) => ({ title: p.id, path: `/code/${p.id}`, ...p })) } }) as Computer;

const remotes = projectRemotes([
  { owner: "acme", name: "e2e", full_name: "acme/e2e", connector_id: "github", role: "tests" },
  { owner: "acme", name: "App", full_name: "acme/App", connector_id: "github", role: "app" },
  { owner: "acme", name: "lab", full_name: "acme/lab", connector_id: "gitlab", role: "app" },
]);
const [e2e, app] = remotes;
const both = [
  { id: "t3-app", repository: "github.com/acme/app" },
  { id: "t3-e2e", repository: "github.com/acme/e2e" },
  { id: "t3-x" },
];

describe("projectCheckout", () => {
  it.each([
    ["a computer that never reported", computer(), [], "unknown"],
    ["the one checkout whose remote is the repository", computer([{ id: "t3-app", repository: "github.com/acme/app" }]), [], "has"],
    ["two checkouts of the repository", computer([{ id: "a", repository: "github.com/acme/app" }, { id: "b", repository: "github.com/acme/app" }]), [], "ambiguous"],
    ["only the tests repository", computer([{ id: "t3-e2e", repository: "github.com/acme/e2e" }]), [], "missing"],
    ["a remote on another host", computer([{ id: "lab", repository: "gitlab.com/acme/lab" }]), [], "missing"],
    ["a picked checkout over the remote", computer([{ id: "a", repository: "github.com/acme/app" }, { id: "b" }]), [{ computer_id: "c-1", repository: "acme/app", harness_project_id: "b" }], "has"],
  ] as const)("%s", (_, c, checkouts, state) => {
    expect(projectCheckout(c, [...checkouts], remotes, app).state).toBe(state);
  });

  it("falls back to the remote when the picked checkout is gone", () => {
    const got = projectCheckout(computer([{ id: "a", repository: "github.com/acme/app" }]), [{ computer_id: "c-1", repository: "acme/app", harness_project_id: "gone" }], remotes, app);
    expect(got).toMatchObject({ state: "has", picked: false, project: { id: "a" } });
  });

  it("counts a pick from before for the repository it clones, or for the app when it clones none", () => {
    const testsPick = [{ computer_id: "c-1", repository: "", harness_project_id: "t3-e2e" }];
    expect(projectCheckout(computer(both), testsPick, remotes, e2e)).toMatchObject({ picked: true, project: { id: "t3-e2e" } });
    expect(projectCheckout(computer(both), testsPick, remotes, app)).toMatchObject({ picked: false, project: { id: "t3-app" } });
    const folderPick = [{ computer_id: "c-1", repository: "", harness_project_id: "t3-x" }];
    expect(projectCheckout(computer(both), folderPick, remotes, app)).toMatchObject({ picked: true, project: { id: "t3-x" } });
    expect(projectCheckout(computer(both), folderPick, remotes, e2e)).toMatchObject({ picked: false, project: { id: "t3-e2e" } });
  });
});

describe("repoCheckouts", () => {
  it("gives each GitHub repository its own checkout", () => {
    expect(repoCheckouts(computer(both), [], remotes).map((r) => [r.repository, r.checkout.state === "has" && r.checkout.project.id])).toEqual([
      ["acme/e2e", "t3-e2e"],
      ["acme/app", "t3-app"],
    ]);
  });

  it("gives a project with no repository one row", () => {
    expect(repoCheckouts(computer(both), [{ computer_id: "c-1", repository: "", harness_project_id: "t3-x" }], [])).toEqual([
      { repository: "", checkout: { state: "has", picked: true, project: { id: "t3-x", title: "t3-x", path: "/code/t3-x" } } },
    ]);
  });
});

describe("withCheckout", () => {
  it("replaces only that repository's pick on that computer", () => {
    const before = [
      { computer_id: "c-1", repository: "acme/app", harness_project_id: "a" },
      { computer_id: "c-1", repository: "acme/e2e", harness_project_id: "e" },
      { computer_id: "c-2", repository: "acme/app", harness_project_id: "x" },
    ];
    expect(withCheckout(before, "c-1", "acme/app", "b")).toEqual([
      { computer_id: "c-1", repository: "acme/e2e", harness_project_id: "e" },
      { computer_id: "c-2", repository: "acme/app", harness_project_id: "x" },
      { computer_id: "c-1", repository: "acme/app", harness_project_id: "b" },
    ]);
  });
});
