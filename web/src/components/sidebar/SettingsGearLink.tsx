import { SettingsIcon } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { UpdateDot } from "@/components/UpdateDot";
import { useSkillsOutdated } from "@/hooks/ComputerSetupHooks";
import { SKILLS_OUTDATED } from "@/models/Pairing";

interface SettingsGearLinkProps {
  collapsed: boolean;
}

// Icon-only in both widths, so it always names itself: beside the rail, or above the open sidebar's account row.
export const SettingsGearLink = ({ collapsed }: SettingsGearLinkProps) => {
  const outdated = useSkillsOutdated();
  const label = outdated ? `Your settings, ${SKILLS_OUTDATED}` : "Your settings";
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button variant="ghost" size="icon" className="relative text-muted-foreground hover:text-foreground" asChild>
          <Link to="/settings" aria-label={label}>
            <SettingsIcon className="size-4" />
            {outdated && <UpdateDot className="top-2 right-2" />}
          </Link>
        </Button>
      </TooltipTrigger>
      <TooltipContent side={collapsed ? "right" : "top"}>{label}</TooltipContent>
    </Tooltip>
  );
};
