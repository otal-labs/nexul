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
  "group/link inline-flex h-7 items-center gap-1.5 rounded-md px-2 text-xs text-muted-foreground transition-colors duration-150 ease-standard first:-ml-2 hover:bg-accent hover:text-foreground";

// One section's reach: its name, the highest level held as glyph and word, and how many of its areas; a link down to it
// when there is something there to read, plain words when the role has no access in it.
export const RoleSectionLink = ({ section, held, top, total }: RoleSectionLinkProps) => (
  <>
    {held === 0 && (
      <span className="inline-flex h-7 items-center gap-1.5 px-2 text-xs text-muted-foreground first:-ml-2">
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
        {top > 0 && <LevelGlyph level={top} />}
        {top > 0 && <span className="text-foreground">{levelName(top)}</span>}
        <span className="font-mono tabular-nums">{`${held}/${total}`}</span>
        <ArrowDown aria-hidden className="size-3 opacity-60 group-hover/link:opacity-100" />
      </a>
    )}
  </>
);
