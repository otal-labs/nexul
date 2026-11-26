import { levelName } from "@/models/PermissionLevel";
import { roleSectionId, type RoleSection } from "@/models/RoleSection";

interface RoleSectionLinkProps {
  section: RoleSection;
  held: number;
  // The highest level the role holds in any of the section's areas.
  top: number;
  total: number;
}

const cellClass = "flex min-w-0 flex-col gap-0.5 rounded-md px-3 py-2";

// A section's reach as a stat cell, the section and its "9/15" over "Up to Delete": a link down to it, or plain words when the role has no access in it.
export const RoleSectionLink = ({ section, held, top, total }: RoleSectionLinkProps) => (
  <>
    {held === 0 && (
      <span className={cellClass}>
        <span className="text-xs text-muted-foreground">{section.title}</span>
        <span className="text-sm text-muted-foreground">No access</span>
      </span>
    )}
    {held > 0 && (
      <a
        href={`#${roleSectionId(section.area)}`}
        onClick={(event) => {
          event.preventDefault();
          document.getElementById(roleSectionId(section.area))?.scrollIntoView({ block: "start" });
        }}
        aria-label={`${section.title}: ${held} of ${total} areas${top > 0 ? `, up to ${levelName(top)}` : ""}`}
        className={`${cellClass} transition-colors duration-150 ease-standard hover:bg-accent`}
      >
        <span aria-hidden className="flex items-baseline gap-2 text-xs text-muted-foreground">
          {section.title}
          <span className="font-mono tabular-nums">{`${held}/${total}`}</span>
        </span>
        <span aria-hidden className="truncate text-sm font-medium text-foreground">{top > 0 ? `Up to ${levelName(top)}` : "Some access"}</span>
      </a>
    )}
  </>
);
