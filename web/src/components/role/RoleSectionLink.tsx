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

// One section's reach as a link down to it: its name, how many of its areas the role holds, and the highest level among them.
export const RoleSectionLink = ({ section, held, top, total }: RoleSectionLinkProps) => (
  <a
    href={`#${roleSectionId(section.area)}`}
    onClick={(event) => {
      event.preventDefault();
      document.getElementById(roleSectionId(section.area))?.scrollIntoView({ block: "start" });
    }}
    aria-label={held > 0 ? `${section.title}: ${held} of ${total} areas, up to ${levelName(top)}` : `${section.title}: no access`}
    className="group/link inline-flex h-7 items-center gap-1.5 rounded-md px-2 text-xs text-muted-foreground transition-colors duration-150 ease-standard first:-ml-2 hover:bg-accent hover:text-foreground"
  >
    {section.title}
    {held > 0 && top > 0 && <LevelGlyph level={top} />}
    {held > 0 && <span className="font-mono text-foreground tabular-nums">{`${held}/${total}`}</span>}
    {held === 0 && <span>No access</span>}
    <ArrowDown aria-hidden className="size-3 opacity-60 group-hover/link:opacity-100" />
  </a>
);
