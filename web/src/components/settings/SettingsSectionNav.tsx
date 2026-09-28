import { Fragment } from "react";
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
  /** Items sharing a group render under one label; a group with no items shows no label. */
  group?: string;
}

interface SettingsSectionNavProps {
  ariaLabel: string;
  items: SettingsSectionNavItem[];
  active: string;
}

// A top row below `lg:` (at 768px a side column left too little width for the content) and a side column from it; shared by every settings-style page so they can't drift.
export const SettingsSectionNav = ({ ariaLabel, items, active }: SettingsSectionNavProps) => (
  <nav aria-label={ariaLabel} className="lg:w-48 lg:shrink-0">
    <ul className="flex items-center gap-1 overflow-x-auto pb-1 lg:flex-col lg:items-stretch lg:gap-0.5 lg:overflow-visible lg:pb-0">
      {items.map((item, index) => (
        <Fragment key={item.section}>
          {item.group !== undefined && item.group !== items[index - 1]?.group && (
            <li
              className={cn(
                "shrink-0 px-3 font-mono text-[11px] font-medium tracking-wide text-muted-foreground/70 uppercase lg:pb-1",
                index > 0 && "ml-2 lg:ml-0 lg:pt-4",
              )}
            >
              {item.group}
            </li>
          )}
          <li className="shrink-0">
            <Link
              to={{ search: `?section=${item.section}` }}
              aria-current={active === item.section ? "page" : undefined}
              className={itemClass(active === item.section, item.danger ?? false)}
            >
              {item.label}
            </Link>
          </li>
        </Fragment>
      ))}
    </ul>
  </nav>
);
