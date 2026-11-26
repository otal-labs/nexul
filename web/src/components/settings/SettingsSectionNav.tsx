import { Fragment } from "react";
import { Link } from "react-router";

import { ActiveIndicator } from "@/components/ActiveIndicator";
import { currentRowFillClass, CurrentRowEdge } from "@/components/CurrentRow";
import { microheaderClass } from "@/components/Microheader";
import { UpdateDot } from "@/components/UpdateDot";
import { cn } from "@/lib/utils";

// The sidebar's row grammar at settings scale: muted at rest, a hover lift, the sliding block marks the current one.
const itemClass = (isActive: boolean) =>
  cn(
    "nav-row inset-focus block rounded-md px-3 py-1.5 text-sm whitespace-nowrap",
    isActive ? "font-medium text-foreground" : "text-muted-foreground hover:bg-accent/60 hover:text-foreground",
  );

export interface SettingsSectionNavItem {
  section: string;
  label: string;
  /** Opens another page instead of a section here (Computers lives on the workspace's Agent page). */
  to?: string | undefined;
  danger?: boolean;
  /** Set to show the update dot beside the label, with this as its screen-reader hint. */
  dot?: string | undefined;
  /** Items sharing a group render under one label; a group with no items shows no label. */
  group?: string | undefined;
}

interface SettingsSectionNavProps {
  ariaLabel: string;
  // The page path each section link hangs off, e.g. /configuration.
  basePath: string;
  items: SettingsSectionNavItem[];
  active: string;
}

// A wrapping top row below `lg:` (at 768px a side column left too little width for the content, and a scrolling row cut its labels) and a side column from it; shared by every settings-style page so they can't drift.
export const SettingsSectionNav = ({ ariaLabel, basePath, items, active }: SettingsSectionNavProps) => (
  <nav aria-label={ariaLabel} className="lg:sticky lg:top-[4.5rem] lg:w-48 lg:shrink-0">
    <ul className="relative isolate flex flex-wrap items-center gap-1 lg:flex-col lg:flex-nowrap lg:items-stretch lg:gap-0.5">
      <ActiveIndicator selector='[aria-current="page"]' className={cn("rounded-md", currentRowFillClass)}>
        <CurrentRowEdge className="hidden lg:block" />
      </ActiveIndicator>
      {items.map((item, index) => (
        <Fragment key={item.section}>
          {/* In the narrow top row a group is a hairline after the last one, not a label that reads as one more item. */}
          {item.group !== undefined && item.group !== items[index - 1]?.group && index > 0 && (
            <li aria-hidden className="mx-2 h-4 w-px shrink-0 bg-border lg:hidden" />
          )}
          {item.group !== undefined && item.group !== items[index - 1]?.group && (
            <li className={cn(microheaderClass, "hidden shrink-0 px-3 lg:block lg:pb-1", index > 0 && "lg:pt-4")}>{item.group}</li>
          )}
          {/* Danger zone is set apart by the same hairline as a group: in the side column above it, in the top row before it. */}
          {item.danger && index > 0 && <li aria-hidden className="mx-2 h-4 w-px shrink-0 bg-border lg:hidden" />}
          <li className={cn("shrink-0", item.danger && index > 0 && "lg:mt-2.5 lg:border-t lg:border-border lg:pt-3")}>
            <Link
              to={item.to ?? `${basePath}/${item.section}`}
              aria-current={active === item.section ? "page" : undefined}
              className={itemClass(active === item.section)}
            >
              {/* Holds the label's medium-weight width, so the current item turning medium never moves its neighbours. */}
              <span
                data-label={item.label}
                className="relative inline-grid after:invisible after:h-0 after:overflow-hidden after:font-medium after:content-[attr(data-label)]"
              >
                {item.label}
                {item.dot && <UpdateDot label={item.dot} className="-top-0.5 -right-2 ring-0" />}
              </span>
            </Link>
          </li>
        </Fragment>
      ))}
    </ul>
  </nav>
);
