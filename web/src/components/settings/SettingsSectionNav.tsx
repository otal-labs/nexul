import { Fragment } from "react";
import { Link } from "react-router";

import { ActiveIndicator } from "@/components/ActiveIndicator";
import { microheaderClass } from "@/components/Microheader";
import { UpdateDot } from "@/components/UpdateDot";
import { cn } from "@/lib/utils";

// The sidebar's row grammar at settings scale: muted at rest, a hover lift, the sliding block marks the current one.
const itemClass = (isActive: boolean) =>
  cn(
    "nav-row block rounded-md px-3 py-1.5 text-sm whitespace-nowrap",
    isActive ? "font-medium text-foreground" : "text-muted-foreground hover:bg-accent/60 hover:text-foreground",
  );

// A deep link can land on an item past the right edge of the narrow top row; bring it into view.
const revealActive = (el: HTMLAnchorElement | null) => el?.scrollIntoView?.({ block: "nearest", inline: "nearest" });

export interface SettingsSectionNavItem {
  section: string;
  label: string;
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

// A top row below `lg:` (at 768px a side column left too little width for the content) and a side column from it; shared by every settings-style page so they can't drift.
export const SettingsSectionNav = ({ ariaLabel, basePath, items, active }: SettingsSectionNavProps) => (
  <nav aria-label={ariaLabel} className="lg:w-48 lg:shrink-0">
    <ul className="relative isolate flex items-center gap-1 overflow-x-auto pb-1 lg:flex-col lg:items-stretch lg:gap-0.5 lg:overflow-visible lg:pb-0">
      <ActiveIndicator selector='[aria-current="page"]' className="rounded-md bg-accent">
        <span className="absolute inset-y-2 left-0 hidden w-0.5 rounded-full bg-brand lg:block" />
      </ActiveIndicator>
      {items.map((item, index) => (
        <Fragment key={item.section}>
          {item.group !== undefined && item.group !== items[index - 1]?.group && (
            <li
              className={cn(
                microheaderClass,
                "shrink-0 px-3 lg:pb-1",
                index > 0 && "ml-2 lg:ml-0 lg:pt-4",
              )}
            >
              {item.group}
            </li>
          )}
          <li className={cn("shrink-0", item.danger && index > 0 && "ml-2 lg:ml-0 lg:mt-3 lg:border-t lg:border-border lg:pt-3")}>
            <Link
              to={`${basePath}/${item.section}`}
              ref={active === item.section ? revealActive : undefined}
              aria-current={active === item.section ? "page" : undefined}
              className={itemClass(active === item.section)}
            >
              <span className="relative">
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
