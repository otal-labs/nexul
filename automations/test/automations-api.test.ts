import { afterAll, beforeAll, describe, expect, test } from "bun:test";
import { createServer } from "node:http";
import type { AddressInfo } from "node:net";
import { httpAutomationsApi } from "../src/automations-api.ts";

// Each credential answers with its own status and body.
const responses: Record<string, [number, string]> = {};
const server = createServer((req, res) => {
  const credential = req.headers.authorization?.replace("Bearer ", "") ?? "";
  const [status, body] = responses[credential] ?? [404, "not found"];
  res.writeHead(status, { "Content-Type": "application/json" }).end(body);
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
    const automations = [{ id: "a1", name: "Ticket finished", token: "dat_h.a1.x" }];
    responses.nxa_live = [200, JSON.stringify({ automations })];
    expect(await httpAutomationsApi.fetchAssignments(url, "nxa_live")).toEqual({ removed: false, automations });
  });
});
