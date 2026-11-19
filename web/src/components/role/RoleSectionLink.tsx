import { ArrowDown } from "lucide-react";

import { LevelGlyph } from "@/components/access/LevelGlyph";
import { levelName } from "@/models/PermissionLevel";
import { roleSectionId, type RoleSection } from "@/models/RoleSection";

interface RoleSectionLinkProps {
  section: RoleSection;
  held: number;
  // The highest level the role holds in any of the section's areas.
  top: number;
  total: number;
}

const linkClass =
  "group/link inline-flex h-7 items-center gap-1.5 rounded-md px-2 text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent hover:text-foreground";

// A section's reach, "Workspace 9/15 Delete", its glyph between count and level: a link down to it, or plain words when the role has no access in it.
export const RoleSectionLink = ({ section, held, top, total }: RoleSectionLinkProps) => (
  <>
    {held === 0 && (
      <span className="inline-flex h-7 items-center gap-1.5 px-2 text-xs text-muted-foreground">
        {section.title}
        <span>No access</span>
      </span>
    )}
    {held > 0 && (
      <a
        href={`#${roleSectionId(section.area)}`}
        onClick={(event) => {
          event.preventDefault();
          document.getElementById(roleSectionId(section.area))?.scrollIntoView({ block: "start" });
        }}
        className={linkClass}
      >
        {section.title}
        <span className="font-mono text-foreground tabular-nums">{`${held}/${total}`}</span>
        {top > 0 && (
          <span>
            <LevelGlyph level={top} className="mr-1 -translate-y-px align-middle" />
            <span className="text-foreground">{levelName(top)}</span>
          </span>
        )}
        <ArrowDown aria-hidden className="size-3 opacity-60 group-hover/link:opacity-100" />
      </a>
    )}
  </>
);
