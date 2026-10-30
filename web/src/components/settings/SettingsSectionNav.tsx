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

// The narrow top row fades on each side that has more to scroll to, so a cut label reads as more, not as half a word.
const markScrolled = (el: HTMLUListElement | null) => {
  if (!el) return;
  const mark = () => {
    el.toggleAttribute("data-scrolled", el.scrollLeft > 0);
    el.toggleAttribute("data-more", el.scrollLeft + el.clientWidth < el.scrollWidth - 1);
  };
  mark();
  el.addEventListener("scroll", mark, { passive: true });
  const resized = typeof ResizeObserver === "undefined" ? undefined : new ResizeObserver(mark);
  resized?.observe(el);
  return () => {
    el.removeEventListener("scroll", mark);
    resized?.disconnect();
  };
};

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
    <ul
      ref={markScrolled}
      className="relative isolate flex items-center gap-1 overflow-x-auto scrollbar-none scroll-px-12 pb-1 max-lg:data-more:[mask-image:linear-gradient(to_right,#000_calc(100%-3rem),transparent)] max-lg:data-scrolled:[mask-image:linear-gradient(to_right,transparent,#000_3rem)] max-lg:data-scrolled:data-more:[mask-image:linear-gradient(to_right,transparent,#000_3rem,#000_calc(100%-3rem),transparent)] lg:flex-col lg:items-stretch lg:gap-0.5 lg:overflow-visible lg:pb-0"
    >
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
              to={`${basePath}/${item.section}`}
              ref={active === item.section ? revealActive : undefined}
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
      {/* Room past the last label, so scrolled to the end it clears the fade. */}
      <li aria-hidden className="w-6 shrink-0 lg:hidden" />
    </ul>
  </nav>
);
