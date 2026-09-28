import { DownloadIcon } from "lucide-react";
import { Link } from "react-router";

import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { INSTANCE_VERSION_SECTION_URL, UpdateChangelog } from "@/components/sidebar/UpdateChangelog";
import { useServerVersion } from "@/hooks/VersionHooks";

interface UpdateButtonProps {
  enabled: boolean;
}

// Hover (or focus) previews what changed; a click or tap goes to the instance version settings to upgrade.
export const UpdateButton = ({ enabled }: UpdateButtonProps) => {
  const { data } = useServerVersion(enabled);

  if (!data?.update_available || !data.latest) return null;

  return (
    <HoverCard openDelay={100} closeDelay={150}>
      <HoverCardTrigger asChild>
        <Link
          to={INSTANCE_VERSION_SECTION_URL}
          aria-label={`Update available: ${data.latest.version}`}
          className="relative flex size-7 items-center justify-center rounded-md text-muted-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/40"
        >
          <DownloadIcon className="size-4" aria-hidden />
          <span className="absolute top-1 right-1 size-1.5 rounded-full bg-warning ring-2 ring-surface-2" aria-hidden />
        </Link>
      </HoverCardTrigger>
      <HoverCardContent side="right" align="start" sideOffset={8} className="w-[min(18rem,calc(100vw-5rem))] p-0">
        <UpdateChangelog current={data.version} latest={data.latest} changes={data.changes} />
      </HoverCardContent>
    </HoverCard>
  );
};
