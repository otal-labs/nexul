import { z } from "zod";

import type { BranchDeployRule } from "@/models/Stack";
import { formatEnv, parseEnv } from "@/lib/env";
import { slugify } from "@/utils/SlugUtility";

// One non-default row of the wizard's deploy branches step; the default branch's row is its deployDefault switch.
const BranchRowShape = z.object({
  pattern: z
    .string()
    .trim()
    .min(1, "Branch is required")
    .regex(/^[^*]*\*?$/, "Use one * at the end, like feature/*"),
  // * marks where the branch goes, like *.example.com.
  hostname: z.string().trim(),
  network: z.string().trim().min(1, "Choose a network"),
  port: z.string().trim(),
  // KEY=value lines, the same text block the env editors use.
  overrides: z.string(),
});

export type BranchRow = z.infer<typeof BranchRowShape>;

export const duplicateHostnameMessage = "Another branch row already uses this hostname pattern";

// A row's hostname only counts on a network with a gateway; elsewhere its field is disabled and never saved.
export const branchRowsFormSchema = (gatewayNetworks: Set<string>) =>
  z
    .object({ deployDefault: z.boolean(), rows: z.array(BranchRowShape) })
    .superRefine((form, ctx) => {
      const seen = new Set<string>();
      form.rows.forEach((row, i) => {
        if (!row.hostname || !gatewayNetworks.has(row.network)) return;
        const path = ["rows", i, "hostname"];
        if (row.pattern.endsWith("*") && !row.hostname.includes("*")) {
          ctx.addIssue({ code: "custom", path, message: "Put * where the branch goes, like *.example.com" });
          return;
        }
        if (!/^[1-9]\d*$/.test(row.port)) {
          ctx.addIssue({ code: "custom", path, message: "Set a port under Advanced options to serve a hostname" });
          return;
        }
        const key = row.hostname.toLowerCase();
        if (seen.has(key)) ctx.addIssue({ code: "custom", path, message: duplicateHostnameMessage });
        seen.add(key);
      });
    });

export type BranchRowsFormData = z.infer<ReturnType<typeof branchRowsFormSchema>>;

// Every non-default row derives its own copy (a wildcard, or a name suffix) so its network and overrides apply.
export const ruleFromRow = (row: BranchRow, served: boolean): BranchDeployRule => {
  const pattern = row.pattern.trim();
  const hostname = served ? row.hostname.trim() : "";
  const overrides = parseEnv(row.overrides);
  return {
    pattern,
    docker_network: row.network,
    ...(!pattern.endsWith("*") && { name_suffix: slugify(pattern) }),
    ...(hostname && { hostname_template: hostname.replaceAll("*", "{branch}"), port: Number(row.port) }),
    ...(Object.keys(overrides).length > 0 && { overrides }),
  };
};

export const rowFromRule = (rule: BranchDeployRule, fallbackPort: string): BranchRow => ({
  pattern: rule.pattern,
  hostname: (rule.hostname_template ?? "").replaceAll("{branch}", "*"),
  network: rule.docker_network,
  port: rule.port ? String(rule.port) : fallbackPort,
  overrides: formatEnv(rule.overrides),
});

// Testers are never sent to a row that reaches production's services: its network with nothing overridden.
export const sharesProduction = (row: Pick<BranchRow, "network" | "overrides">, productionNetwork: string): boolean =>
  row.network === productionNetwork && Object.keys(parseEnv(row.overrides)).length === 0;
