import { screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useWorkspaceStore } from "@/stores/workspaceStore";
import { renderAgent } from "@/test/agentComputers";
import { asViewer } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: (e: Error) => e.message }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const play = (id: string, label: string, needs: string[], overrides: Record<string, unknown> = {}) => ({
  id, workspace_id: "ws-1", label, type: "ticket", description: "", instructions: "", enabled: true, show_when_stage: "progress",
  excluded_project_ids: [], repository: "app", needs, builtin_key: "", created_by: "", created_at: "", updated_at: "", can_run: true, ...overrides,
});

const computer = (id: string, name: string) => ({ id, name, token_expires_at: "2099-01-01T00:00:00Z", position: 0, runs_at_once: 1, running: 0, chat_project_id: "" });

const PLAYS = [
  play("p-fix", "Fix with AI", []),
  play("p-test", "Test with AI", []),
  play("p-win", "Test on Windows", ["windows-vm"]),
  play("p-off", "Switched off", ["gpu"], { enabled: false }),
  play("p-skip", "Skips this project", ["gpu"], { excluded_project_ids: ["proj-1"] }),
];

const placementReads = () => mocks.get.mock.calls.filter((call) => String(call[0]).includes("/placement"));

beforeEach(() => {
  mocks.get.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
  mocks.get.mockImplementation(
    asViewer({ role_name: "Owner", roles: [{ id: "r", name: "Owner", is_owner: true }], permissions: ["plays:read"] }, (url: string) => {
      if (url === "/api/pairing/computers") return { computers: [computer("c-desk", "desk"), computer("c-win", "winbox")] };
      if (url === "/api/pairing/presence") return { computers: {} };
      if (url === "/api/runners/enrollments") return [];
      if (url === "/api/workspaces") return [{ id: "ws-1", name: "Acme", slug: "acme" }];
      if (url === "/api/projects") return [{ id: "proj-1", name: "Web", prefix: "WEB", position: 0 }];
      if (url === "/api/pairing/projects") return { links: [] };
      if (url === "/api/pairing/defaults") return {};
      if (url === "/api/workspaces/ws-1/plays") return PLAYS;
      if (url.startsWith("/api/pairing/projects/proj-1/placement")) {
        return {
          answers: [
            { needs: [], computer_id: "c-desk", passed_over: [] },
            { needs: ["windows-vm"], computer_id: "c-win", passed_over: [{ computer_id: "c-desk", name: "desk", why: "missing_need", text: "desk doesn't have windows-vm" }] },
          ],
        };
      }
      return [];
    }),
  );
});

describe("where a project's plays land now", () => {
  it("answers every play from one read, plays sharing needs sharing an answer, switched-off and excluded plays left out", async () => {
    renderAgent("/acme/agent/projects/proj-1");

    const section = await screen.findByRole("region", { name: "Where its plays land now" });
    const rows = await within(section).findAllByRole("listitem");
    expect(rows.map((row) => row.textContent)).toEqual([
      "Fix with AI, Test with AIdesk",
      "Test on Windowswindows-vmwinboxPassed over: desk doesn't have windows-vm",
    ]);
    expect(placementReads()).toHaveLength(1);
    expect(placementReads()[0]?.[0]).toBe("/api/pairing/projects/proj-1/placement?needs=&needs=windows-vm");
  });
});
