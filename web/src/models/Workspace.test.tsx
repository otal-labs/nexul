import { describe, expect, it } from "vitest";

import type { RouteArea } from "@/models/Access";
import {
  normalizeSlugInput,
  replaceWorkspaceSlug,
  slugify,
  switchWorkspacePath,
  WorkspaceGeneralFormSchema,
  type WorkspaceAccess,
} from "@/models/Workspace";

const access = (areas: RouteArea[], configurationSections: string[] = ["roles", "plays", "danger"]): WorkspaceAccess => ({
  canOpen: (area) => areas.includes(area),
  configurationSections,
});

const everything = access(["tickets", "docs", "memories", "runners", "topology", "automations", "newProject"]);

describe("switchWorkspacePath", () => {
  it.each([
    ["/a", "/b"],
    ["/a/board/ONLY", "/b/board"],
    ["/a/board/ONLY/extra", "/b/board"],
    ["/a/tickets/ONLY-12", "/b/board"],
    ["/a/projects/ONLY/settings/general", "/b/board"],
    ["/a/projects/ONLY/interview", "/b/board"],
    ["/a/docs/ONLY/d-1", "/b/docs"],
    ["/a/memories/ONLY/m-1", "/b/memories"],
    ["/a/chat/c-1", "/b/chat"],
    ["/a/inbox", "/b/inbox"],
    ["/a/runners", "/b/runners"],
    ["/a/topology", "/b/topology"],
    ["/a/stacks/s-1/deploys/d-1", "/b/topology"],
    ["/a/automations/au-1", "/b/automations"],
    ["/a/wizard/project/service?project=p-1", "/b/wizard/project/project"],
    ["/a/configuration/plays", "/b/configuration/plays"],
    ["/a/nowhere", "/b"],
  ])("%s lands on %s", (from, to) => {
    expect(switchWorkspacePath(from, "b", everything)).toBe(to);
  });

  it("lands on the first Configuration section the viewer has there when theirs is hidden", () => {
    expect(switchWorkspacePath("/a/configuration/mentions", "b", everything)).toBe("/b/configuration/roles");
  });

  it("goes home when Configuration has nothing but Danger zone there", () => {
    expect(switchWorkspacePath("/a/configuration/roles", "b", access([], ["danger"]))).toBe("/b");
  });

  it.each(["/a/board/ONLY", "/a/docs", "/a/runners", "/a/stacks/s-1", "/a/wizard/project/project"])(
    "goes home from %s when the viewer can't open that section there",
    (from) => {
      expect(switchWorkspacePath(from, "b", access([]))).toBe("/b");
    },
  );
});

describe("slugify", () => {
  it.each([
    ["Rixwave Labs!", "rixwave-labs"],
    ["  OTAL  ", "otal"],
    ["Café Crème", "caf-cr-me"],
    ["!!!", "workspace"],
  ])("%s → %s, as the server derives it", (name, slug) => {
    expect(slugify(name)).toBe(slug);
  });
});

describe("replaceWorkspaceSlug", () => {
  it.each([
    ["/acme", "/acme-labs"],
    ["/acme/board/WEB", "/acme-labs/board/WEB"],
    ["/acme-labs/board", undefined],
    ["/acmeboard", undefined],
    ["/settings/profile", undefined],
  ])("%s becomes %s", (pathname, want) => {
    expect(replaceWorkspaceSlug(pathname, "acme", "acme-labs")).toBe(want);
  });
});

describe("normalizeSlugInput", () => {
  it.each([
    ["My App", "my-app"],
    ["rix_wave!!", "rix-wave-"],
    ["--lead", "lead"],
    ["a   b", "a-b"],
  ])("types %s as %s", (typed, kept) => {
    expect(normalizeSlugInput(typed)).toBe(kept);
  });
});

describe("WorkspaceGeneralFormSchema", () => {
  const slugError = (slug: string) => {
    const result = WorkspaceGeneralFormSchema.safeParse({ name: "Acme", slug });
    return result.success ? undefined : result.error.issues[0]?.message;
  };

  it.each(["rixwave", "acme-2", "a"])("accepts %s", (slug) => {
    expect(slugError(slug)).toBeUndefined();
  });

  it.each([
    ["", "URL is required"],
    ["acme-", "Use lowercase letters and digits, joined by single dashes"],
    ["ac--me", "Use lowercase letters and digits, joined by single dashes"],
    ["a".repeat(49), "URL can be at most 48 characters"],
    ["settings", "That address is used by the app itself"],
    ["wizard", "That address is used by the app itself"],
  ])("refuses %s", (slug, message) => {
    expect(slugError(slug)).toBe(message);
  });
});
