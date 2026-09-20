import { Link } from "react-router";

import { cn } from "@/lib/utils";

const itemClass = (isActive: boolean, danger: boolean) =>
  cn(
    "block rounded-md px-3 py-1.5 text-sm whitespace-nowrap transition-colors duration-150 ease-standard",
    danger && "text-destructive",
    isActive && (danger ? "bg-destructive/10 font-medium" : "bg-accent font-medium text-foreground"),
    !isActive &&
      (danger ? "hover:bg-destructive/10" : "text-muted-foreground hover:bg-accent/60 hover:text-foreground"),
  );

export interface SettingsSectionNavItem {
  section: string;
  label: string;
  danger?: boolean;
}

interface SettingsSectionNavProps {
  ariaLabel: string;
  items: SettingsSectionNavItem[];
  active: string;
}

// Collapses to a horizontal tab row below `md:`; shared by both settings pages so they can't drift.
export const SettingsSectionNav = ({ ariaLabel, items, active }: SettingsSectionNavProps) => (
  <nav aria-label={ariaLabel} className="md:w-48 md:shrink-0">
    <ul className="flex gap-1 overflow-x-auto pb-1 md:flex-col md:gap-0.5 md:overflow-visible md:pb-0">
      {items.map((item) => (
        <li key={item.section} className="shrink-0">
          <Link
            to={{ search: `?section=${item.section}` }}
            aria-current={active === item.section ? "page" : undefined}
            className={itemClass(active === item.section, item.danger ?? false)}
          >
            {item.label}
          </Link>
        </li>
      ))}
    </ul>
  </nav>
);
