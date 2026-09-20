import { useState } from "react";

import type { GroupedDiscovery, ImportRequest } from "@/models/Machine";

interface Selection {
  stacks: Record<string, Set<string>>;
  standalone: Set<string>;
  gateways: Set<string>;
}

const emptySelection: Selection = { stacks: {}, standalone: new Set(), gateways: new Set() };

const toggled = (names: Iterable<string>, name: string): Set<string> => {
  const next = new Set(names);
  if (next.has(name)) {
    next.delete(name);
    return next;
  }
  next.add(name);
  return next;
};

// Selection state for the import door's checklist (spec §8). Local UI state, not server state — it only ever
// reflects what the owner has ticked in the current discovery report.
export const useImportSelection = () => {
  const [selection, setSelection] = useState<Selection>(emptySelection);

  // Ticked by default (issue 08 answer: "everything is ticked by default except containers a Nexul stack
  // already manages"); the discovery report already leaves out tracked containers, so nothing to skip here.
  const seed = (report: GroupedDiscovery) => {
    const stacks: Record<string, Set<string>> = {};
    for (const group of report.stacks) stacks[group.project] = new Set(group.containers.map((c) => c.name));
    setSelection({
      stacks,
      standalone: new Set(report.standalone.map((c) => c.name)),
      gateways: new Set(report.gateways.map((c) => c.name)),
    });
  };

  const toggleContainer = (project: string, name: string) =>
    setSelection((prev) => {
      const next = toggled(prev.stacks[project] ?? [], name);
      return { ...prev, stacks: { ...prev.stacks, [project]: next } };
    });

  const toggleInSet = (key: "standalone" | "gateways", name: string) =>
    setSelection((prev) => {
      const next = toggled(prev[key], name);
      return { ...prev, [key]: next };
    });

  const toRequest = (projectId: string): ImportRequest => ({
    project_id: projectId,
    stacks: Object.entries(selection.stacks)
      .filter(([, names]) => names.size > 0)
      .map(([project, names]) => ({ project, containers: [...names] })),
    standalone: [...selection.standalone],
    gateways: [...selection.gateways],
  });

  return {
    selection,
    seed,
    toggleContainer,
    toggleStandalone: (name: string) => toggleInSet("standalone", name),
    toggleGateway: (name: string) => toggleInSet("gateways", name),
    toRequest,
  };
};
