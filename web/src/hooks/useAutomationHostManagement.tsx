import type { EntityAnswer } from "@nexul/client-core/actionState";
import { hasPermission } from "@nexul/client-core/permissions";

import { useIsInstanceOwner } from "@/hooks/AccessHooks";
import { useFetchMachines } from "@/hooks/RunnerHooks";

// Adding a host takes Manage on the machine it joins, so it is offered to whoever Manages one machine or is the instance Owner.
export const useAutomationHostManagement = (): { isOwner: boolean; managed: string[]; answer: EntityAnswer } => {
  const { data: machines } = useFetchMachines();
  const isOwner = useIsInstanceOwner();
  const managed = (machines ?? []).filter((machine) => hasPermission(machine.actions, "machines:write")).map((machine) => machine.name);
  return { isOwner, managed, answer: { allowed: isOwner || managed.length > 0, source: { sentence: "" } } };
};
