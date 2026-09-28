import { SettingsIcon } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";

export const SettingsGearLink = () => (
  <Button variant="ghost" size="icon" asChild>
    <Link to="/settings" aria-label="Your settings" title="Your settings">
      <SettingsIcon className="size-4" />
    </Link>
  </Button>
);
