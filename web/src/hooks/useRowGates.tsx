import type { ActionGate } from "@nexul/client-core/actionState";

import type { GatedRowAction } from "@/components/listpane/rowMenu";
import { useActionStates } from "@/hooks/useActionState";

type Gates = Partial<Record<GatedRowAction | "onDelete", ActionGate>>;

// `offered` keeps a row's handler while its action shows at all, live or locked, so RowMenu can draw it with its gate.
export const useRowGates = (gates: Gates) => {
  const { onDelete, ...rest } = gates;
  const actions = Object.keys(gates) as (GatedRowAction | "onDelete")[];
  const states = useActionStates(actions.map((action) => gates[action] as ActionGate));
  return {
    offered: (action: GatedRowAction | "onDelete"): boolean => {
      const state = states[actions.indexOf(action)];
      return !!state && state.state !== "hidden";
    },
    gates: rest,
    deleteGate: onDelete,
  };
};
