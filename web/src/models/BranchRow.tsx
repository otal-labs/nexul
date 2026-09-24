import { z } from "zod";

import type { BranchDeployRule } from "@/models/Stack";
import { formatEnv, parseEnv } from "@/lib/env";
import { slugify } from "@/utils/SlugUtility";

// One non-default row of the wizard's deploy branches step; the default branch's row is fixed, not a form row.
export const BranchRowSchema = z
  .object({
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
  })
  .superRefine((row, ctx) => {
    if (!row.hostname) return;
    if (row.pattern.endsWith("*") && !row.hostname.includes("*")) {
      ctx.addIssue({ code: "custom", path: ["hostname"], message: "Put * where the branch goes, like *.example.com" });
      return;
    }
    if (!/^[1-9]\d*$/.test(row.port)) {
      ctx.addIssue({ code: "custom", path: ["hostname"], message: "Set a port under Advanced options to serve a hostname" });
    }
  });

export const BranchRowsFormSchema = z.object({ rows: z.array(BranchRowSchema) });

export type BranchRow = z.infer<typeof BranchRowSchema>;
export type BranchRowsFormData = z.infer<typeof BranchRowsFormSchema>;

// Every non-default row derives its own copy (a wildcard, or a name suffix) so its network and overrides apply.
export const ruleFromRow = (row: BranchRow): BranchDeployRule => {
  const pattern = row.pattern.trim();
  const hostname = row.hostname.trim();
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
