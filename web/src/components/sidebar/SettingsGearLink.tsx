import { SettingsIcon } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { UpdateDot } from "@/components/UpdateDot";
import { useSkillsOutdated } from "@/hooks/ComputerSetupHooks";
import { SKILLS_OUTDATED } from "@/models/Pairing";

export const SettingsGearLink = () => {
  const outdated = useSkillsOutdated();
  const label = outdated ? `Your settings, ${SKILLS_OUTDATED}` : "Your settings";
  return (
    <Button variant="ghost" size="icon" className="relative" asChild>
      <Link to="/settings" aria-label={label} title={label}>
        <SettingsIcon className="size-4" />
        {outdated && <UpdateDot className="top-2 right-2" />}
      </Link>
    </Button>
  );
};
