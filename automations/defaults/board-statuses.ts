import type { Ctx } from "@otal-labs/nexul-sdk/context";

type StatusKind = "backlog" | "progress" | "review" | "testing" | "done";

// Narrows the status list to the fields read, since the response is not typed.
function statusFields(raw: unknown): { id: string; kind: string; position: number }[] {
  if (!Array.isArray(raw)) return [];
  return raw.flatMap((s: unknown) => {
    const { id, kind, position } = (s ?? {}) as { id?: unknown; kind?: unknown; position?: unknown };
    return typeof id === "string" && typeof kind === "string" && typeof position === "number" ? [{ id, kind, position }] : [];
  });
}

// Statuses are per project, so a default finds its target by kind: the first column of that kind on the board.
export async function firstStatusOfKind(ctx: Ctx, projectId: string, kind: StatusKind): Promise<string | undefined> {
  const raw = await ctx.api.request<unknown>("GET", `/api/statuses?project_id=${encodeURIComponent(projectId)}`);
  const matches = statusFields(raw).filter((s) => s.kind === kind);
  return matches.sort((a, b) => a.position - b.position)[0]?.id;
}
