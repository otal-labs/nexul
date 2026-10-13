import { SparklesIcon } from "lucide-react";
import { Link } from "react-router";

import { WELCOME_STEP_NAMES } from "@nexul/client-core/welcome";

import { MenuSeparator, menuItemClass } from "@/components/MenuItem";
import { useWelcomeProgress } from "@/hooks/WelcomeHooks";
import { cn } from "@/lib/utils";
import { welcomeMenuSummary } from "@/utils/WelcomeMenuUtility";

interface WelcomeMenuSectionProps {
  onClose: () => void;
}

// The account menu's first item: Welcome and what is left, then a row per step not done that opens the wizard on it.
export const WelcomeMenuSection = ({ onClose }: WelcomeMenuSectionProps) => {
  const { data: progress } = useWelcomeProgress();

  if (!progress || progress.steps.length === 0) {
    return null;
  }

  const { open, left, done, total } = welcomeMenuSummary(progress);

  return (
    <>
      <Link to="/wizard/welcome" className={menuItemClass} onClick={onClose}>
        <SparklesIcon aria-hidden />
        <span className="flex-1">Welcome</span>
        {left > 0 && <span className="size-1.5 rounded-full bg-destructive" aria-hidden />}
        <span className="font-mono text-xs tabular-nums text-muted-foreground">{left > 0 ? `${left} left` : `${done}/${total}`}</span>
      </Link>
      {open.map((step) => (
        <Link key={step.key} to={`/wizard/welcome?step=${step.key}`} className={cn(menuItemClass, "pl-8.5")} onClick={onClose}>
          <span className="flex-1 truncate text-muted-foreground">{WELCOME_STEP_NAMES[step.key]}</span>
          {step.state === "skipped" && <span className="font-mono text-[11px] text-muted-foreground/80">Skipped</span>}
        </Link>
      ))}
      <MenuSeparator />
    </>
  );
};
