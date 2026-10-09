import { DownloadIcon } from "lucide-react";
import { Link } from "react-router";

import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { INSTANCE_VERSION_SECTION_URL, UpdateChangelog } from "@/components/sidebar/UpdateChangelog";
import { UpdateDot } from "@/components/UpdateDot";
import { useServerVersion } from "@/hooks/VersionHooks";
import { cn } from "@/lib/utils";

interface UpdateButtonProps {
  className?: string;
}

// Hover (or focus) previews what changed; a click or tap goes to the instance version settings to upgrade.
export const UpdateButton = ({ className }: UpdateButtonProps) => {
  const { data } = useServerVersion();

  if (!data?.update_available || !data.latest) return null;

  return (
    <HoverCard openDelay={100} closeDelay={150}>
      <HoverCardTrigger asChild>
        <Link
          to={INSTANCE_VERSION_SECTION_URL}
          aria-label={`Update available: ${data.latest.version}`}
          className={cn(
            "relative flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent/60 hover:text-foreground",
            className,
          )}
        >
          <DownloadIcon className="size-4" aria-hidden />
          <UpdateDot />
        </Link>
      </HoverCardTrigger>
      <HoverCardContent side="right" align="start" sideOffset={8} className="w-[min(18rem,calc(100vw-5rem))] p-0">
        <UpdateChangelog current={data.version} latest={data.latest} changes={data.changes} />
      </HoverCardContent>
    </HoverCard>
  );
};
