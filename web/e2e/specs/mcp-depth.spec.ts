import { describe, expect, it } from "vitest";

import { E2E_PROJECT_ID, MCP_URL, makeApiClient, mcpHeaders, mintToken } from "../helpers";

const TOKEN = mintToken();

const rpc = async <T = unknown>(method: string, params: unknown, id = 1) => {
  const res = await fetch(MCP_URL, {
    method: "POST",
    headers: mcpHeaders(TOKEN),
    body: JSON.stringify({ jsonrpc: "2.0", id, method, params }),
  });
  return { status: res.status, body: (await res.json()) as T };
};

const api = makeApiClient(TOKEN);

const json = <T = unknown>(body: T) => ({ body: JSON.stringify(body) });

type ToolCallResponse = {
  result?: { content?: { type: string; text?: string }[] };
  error?: { message?: string };
};

const callTool = async (name: string, arguments_: Record<string, unknown>) => {
  const res = await rpc<ToolCallResponse>("tools/call", { name, arguments: arguments_ });
  expect(res.status).toBe(200);
  return res.body;
};

describe("mcp tools + resources (depth)", () => {
  it("creates, searches, reads a doc — then reads it as a resource", async () => {
    const marker = `mcp-e2e-${Date.now()}`;
    const title = `${marker} runbook`;

    const created = await callTool("doc_create", {
      project_id: E2E_PROJECT_ID,
      title,
      body: `# ${marker}\n\nBody written through MCP.`,
    });
    expect(created.error).toBeUndefined();
    const doc = JSON.parse(created.result!.content![0].text!) as { id: string; title: string; body: string };
    expect(doc.title).toBe(title);
    expect(doc.body).toContain(marker);

    try {
      // doc_list with a query finds the doc by its unique marker; the FTS index updates on the doc.created event, so poll briefly.
      let hit = false;
      for (let attempt = 0; attempt < 10 && !hit; attempt++) {
        const searched = await callTool("doc_list", { query: marker, limit: 5 });
        expect(searched.error).toBeUndefined();
        const page = JSON.parse(searched.result!.content![0].text!) as { items: { id: string }[] };
        hit = page.items.some((h) => h.id === doc.id);
        if (!hit) await new Promise((resolve) => setTimeout(resolve, 300));
      }
      expect(hit).toBe(true);

      const fetched = await callTool("doc_get", { id: doc.id });
      expect(fetched.error).toBeUndefined();
      expect(fetched.result!.content![0].text).toContain(marker);

      // resources/templates/list advertises the docs template, and resources/read serves the doc back as markdown.
      const listed = await rpc<{
        result?: { resourceTemplates?: { uriTemplate: string }[] };
      }>("resources/templates/list", {});
      expect(listed.status).toBe(200);
      const templates = listed.body.result?.resourceTemplates?.map((t) => t.uriTemplate) ?? [];
      expect(templates).toContain("docs://{id}");

      const read = await rpc<{ result?: { contents?: { uri: string; text: string }[] } }>(
        "resources/read",
        { uri: `docs://${doc.id}` },
      );
      expect(read.status).toBe(200);
      const contents = read.body.result?.contents ?? [];
      expect(contents.some((c) => c.uri === `docs://${doc.id}` && c.text.includes(marker))).toBe(true);
    } finally {
      await api(`/api/docs/${doc.id}`, { method: "DELETE" });
    }
  });

  it("reads a ticket through the tickets:// resource", async () => {
    const marker = `mcp-ticket-${Date.now()}`;
    const ticket = await api<{ id: string }>("/api/tickets", {
      method: "POST",
      ...json({ title: marker, body: "ticket body", project_id: "e2e00000-0000-4000-8000-000000000010" }),
    });
    expect(ticket.status).toBe(201);

    try {
      const read = await rpc<{ result?: { contents?: { uri: string; text: string }[] } }>(
        "resources/read",
        { uri: `tickets://${ticket.body!.id}` },
      );
      expect(read.status).toBe(200);
      const contents = read.body.result?.contents ?? [];
      expect(contents.some((c) => c.text.includes(marker))).toBe(true);
    } finally {
      await api(`/api/tickets/${ticket.body!.id}`, { method: "DELETE" });
    }
  });
});
