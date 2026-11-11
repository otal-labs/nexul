import { describe, expect, test } from "bun:test";
import type { AutomationTarget, AutomationsApi } from "../src/automations-api.ts";
import { pollOnce, uninstallSelf } from "../src/host.ts";

const target: AutomationTarget = { id: "a1", name: "Ticket finished", token: "dat_h.a1.x", workspace_id: "ws-1", workspace: "acme" };

function deps(result: Awaited<ReturnType<AutomationsApi["fetchAssignments"]>>) {
  const reconciled: AutomationTarget[][] = [];
  const credentials: string[] = [];
  const reports: string[] = [];
  return {
    reconciled,
    credentials,
    reports,
    poll: {
      api: {
        fetchAssignments: async (_url: string, credential: string) => {
          credentials.push(credential);
          return result;
        },
        fetchState: async () => {
          throw new Error("not used");
        },
        reportManagers: async () => {},
      },
      managers: {
        report: async () => {
          reports.push("reported");
        },
        servable: (targets: AutomationTarget[]) => targets,
      },
      supervisor: {
        reconcile: async (targets: AutomationTarget[]) => {
          reconciled.push(targets);
        },
      },
      serverUrl: "http://server",
      credential: "nxa_1",
    },
  };
}

describe("pollOnce", () => {
  test("a network failure surfaces without touching the workers", async () => {
    const d = deps({ removed: false, automations: [] });
    d.poll.api.fetchAssignments = async () => {
      throw new Error("connection refused");
    };

    await expect(pollOnce(d.poll)).rejects.toThrow("connection refused");
    expect(d.reconciled).toHaveLength(0);
  });

  test("a removed host stops reconciling and says so", async () => {
    const d = deps({ removed: true });

    expect(await pollOnce(d.poll)).toBe("removed");
    expect(d.reconciled).toHaveLength(0);
    expect(d.reports).toHaveLength(0);
  });

  test("reconciles the workers to the assignments, polled with the host credential", async () => {
    const d = deps({ removed: false, automations: [target] });

    expect(await pollOnce(d.poll)).toBe("running");
    expect(d.credentials).toEqual(["nxa_1"]);
    expect(d.reconciled).toEqual([[target]]);
    expect(d.reports).toEqual(["reported"]);
  });
});

describe("uninstallSelf", () => {
  test("detaches the uninstall through nexul, and survives it failing", () => {
    const calls: [string, string[]][] = [];
    uninstallSelf("/usr/local/bin/nexul", "jobs", (command, args) => {
      calls.push([command, args]);
      return 1;
    });

    expect(calls).toEqual([["/usr/local/bin/nexul", ["uninstall", "automations", "jobs", "--detach"]]]);
  });
});
