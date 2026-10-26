import { HourglassIcon, ShieldCheck } from "lucide-react";

import { RailTooltip } from "@/components/sidebar/RailTooltip";
import { SetupNavLink } from "@/components/sidebar/SetupNavLink";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";

interface AccessSetupNavProps {
  setup: "owner" | "member";
  collapsed: boolean;
}

const row = "flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm";

// While the one-time access setup is open: Set up access gets an Owner back to it, and a member sees why no project shows.
export const AccessSetupNav = ({ setup, collapsed }: AccessSetupNavProps) => {
  const wsPath = useWorkspacePath();
  return (
    <>
      {setup === "owner" && (
        <SetupNavLink to={wsPath("/access")} label="Set up access" icon={ShieldCheck} collapsed={collapsed} />
      )}
      {setup === "member" && (
        <RailTooltip label="Projects being set up" collapsed={collapsed}>
          <p className={cn(row, "text-muted-foreground")} aria-label={collapsed ? "Projects being set up" : undefined}>
            <span className="flex w-8 shrink-0 justify-center">
              <HourglassIcon className="size-4" aria-hidden />
            </span>
            {!collapsed && <span className="min-w-0 flex-1 truncate text-left">Projects being set up</span>}
          </p>
        </RailTooltip>
      )}
    </>
  );
};
