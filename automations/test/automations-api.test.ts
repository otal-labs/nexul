import { afterAll, beforeAll, describe, expect, test } from "bun:test";
import { createServer } from "node:http";
import type { AddressInfo } from "node:net";
import { httpAutomationsApi } from "../src/automations-api.ts";

// Each credential answers with its own status and body, or per path as "<credential> <path>".
const responses: Record<string, [number, string]> = {};
const requested: string[] = [];
const posted: string[] = [];
const server = createServer((req, res) => {
  const credential = req.headers.authorization?.replace("Bearer ", "") ?? "";
  requested.push(req.url ?? "");
  req.on("data", (chunk) => posted.push(String(chunk)));
  req.on("end", () => {
    const [status, body] = responses[`${credential} ${req.url}`] ?? responses[credential] ?? [404, "not found"];
    res.writeHead(status, { "Content-Type": "application/json" }).end(body);
  });
});
let url = "";
beforeAll(async () => {
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  url = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
});
afterAll(() => server.close());

describe("httpAutomationsApi.fetchAssignments", () => {
  test("a plain 401 is an error, not a removal", async () => {
    responses.nxa_unknown = [401, '{"message":"unauthorized","code":"UNAUTHORIZED"}'];
    await expect(httpAutomationsApi.fetchAssignments(url, "nxa_unknown")).rejects.toThrow("401");
  });

  test("the removed refusal means this host was removed", async () => {
    responses.nxa_removed = [401, '{"error":"automations_host_removed"}'];
    expect(await httpAutomationsApi.fetchAssignments(url, "nxa_removed")).toEqual({ removed: true });
  });

  test("returns the automations placed on this host", async () => {
    const automations = [{ id: "a1", name: "Ticket finished", token: "dat_h.a1.x", workspace: "acme" }];
    responses.nxa_live = [200, JSON.stringify({ automations })];
    expect(await httpAutomationsApi.fetchAssignments(url, "nxa_live")).toEqual({ removed: false, automations });
    expect(requested).toContain("/api/automation-hosts/self/assignments?version=dev");
  });
});

describe("httpAutomationsApi.fetchState", () => {
  test("a write that leaves the version and config alone, like the worker's own announce, reads as the same state", async () => {
    const target = { id: "a1", name: "PR opened", token: "dat_h.a1.x", workspace: "acme" };
    const diff = JSON.stringify({ active: { id: "v1", code: "export default {}" } });
    responses["dat_h.a1.x /api/automations/a1/versions/diff"] = [200, diff];
    const automation = (updatedAt: string) => JSON.stringify({ enabled: true, updated_at: updatedAt, config_values: { channel: "ops" } });

    responses["dat_h.a1.x /api/automations/a1"] = [200, automation("2026-11-09T02:04:09Z")];
    const before = await httpAutomationsApi.fetchState(target, url);
    responses["dat_h.a1.x /api/automations/a1"] = [200, automation("2026-11-09T02:04:10Z")];
    const after = await httpAutomationsApi.fetchState(target, url);

    expect(after).toEqual(before);
    expect(before.configValues).toBe('{"channel":"ops"}');
  });
});

describe("httpAutomationsApi.reportManagers", () => {
  test("posts the managers to the host's own route with its credential", async () => {
    responses.nxa_live = [204, ""];
    await httpAutomationsApi.reportManagers(url, "nxa_live", [{ workspace: "acme", manager: "op", state: "ready" }]);
    expect(requested).toContain("/api/automation-hosts/self/managers");
    expect(posted.map((b) => JSON.parse(b))).toContainEqual({ managers: [{ workspace: "acme", manager: "op", state: "ready" }] });
  });

  test("a refusal is an error, so the report is tried again", async () => {
    responses.nxa_refused = [401, "{}"];
    await expect(httpAutomationsApi.reportManagers(url, "nxa_refused", [])).rejects.toThrow("401");
  });
});
