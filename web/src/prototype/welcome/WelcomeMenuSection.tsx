import { Sparkles } from "lucide-react";
import { useNavigate } from "react-router";

import { MenuSeparator, menuItemClass } from "@/components/MenuItem";
import { cn } from "@/lib/utils";

import { leftCount, STEPS, useWelcomeProto } from "./model";

// The account menu's top item: Welcome and its count, then one row per step that is not done; a skipped one says so.
export const WelcomeMenuSection = ({ onClose }: { onClose: () => void }) => {
  const navigate = useNavigate();
  const order = useWelcomeProto((s) => s.order);
  const states = useWelcomeProto((s) => s.states);
  const open = order.filter((k) => states[k] !== "done");
  const left = leftCount(order, states);
  const go = (step?: string) => {
    onClose();
    navigate(`/wizard/welcome${step ? `?step=${step}` : ""}`);
  };

  return (
    <>
      <button type="button" className={menuItemClass} onClick={() => go()}>
        <Sparkles aria-hidden />
        <span className="flex-1">Welcome</span>
        {left > 0 && <span className="size-1.5 rounded-full bg-destructive" aria-hidden />}
        <span className="font-mono text-xs text-muted-foreground tabular-nums">
          {left > 0 ? `${left} left` : `${order.length - open.length}/${order.length}`}
        </span>
      </button>
      {open.map((k) => (
        <button key={k} type="button" className={cn(menuItemClass, "pl-8.5")} onClick={() => go(k)}>
          <span className="flex-1 truncate text-muted-foreground">{STEPS[k].name}</span>
          {states[k] === "skipped" && <span className="font-mono text-[11px] text-muted-foreground/80">Skipped</span>}
        </button>
      ))}
      <MenuSeparator />
    </>
  );
};

export const useWelcomeLeft = () => useWelcomeProto((s) => leftCount(s.order, s.states));
