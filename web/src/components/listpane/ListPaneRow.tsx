import { useEffect, useRef, type ReactNode } from "react";
import { Link } from "react-router";

import { RowActions } from "@/components/listpane/RowActions";
import { offersMenu, type RowMenu } from "@/components/listpane/rowMenu";
import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { cn } from "@/lib/utils";

interface ListPaneRowProps {
  to: string;
  title: string;
  /** A small kind icon before the title, such as a doc's page. */
  leading?: ReactNode;
  /** A small state icon after the title, such as a lock. */
  titleIcon?: ReactNode;
  snippet?: string;
  selected: boolean;
  /** Time, avatar, chips: the action cluster takes their place on hover, focus, and selection. */
  meta?: ReactNode;
  actions?: ReactNode;
  /** The row's menu: its ⋯ and its right-click menu hold the same items. */
  menu?: RowMenu | undefined;
  /** Always visible at the right edge, e.g. a switch. */
  trailing?: ReactNode;
}

// The link's ::after covers the row so the whole row opens it; actions and trailing sit above that layer.
export const ListPaneRow = ({ to, title, leading, titleIcon, snippet = "", selected, meta, actions, menu, trailing }: ListPaneRowProps) => {
  const ref = useRef<HTMLLIElement>(null);
  const hasActions = !!actions || offersMenu(menu);

  // A deep link can select a row far down the list.
  useEffect(() => {
    if (selected) ref.current?.scrollIntoView?.({ block: "nearest" });
  }, [selected]);

  return (
    <RowContextMenu menu={menu}>
      <li
        ref={ref}
        className={cn(
          "group relative flex items-center gap-2 border-l-2 px-3 transition-colors duration-150 ease-standard",
          snippet === "" ? "min-h-10" : "min-h-14",
          selected ? "border-l-brand bg-accent" : "border-l-transparent hover:bg-accent/40",
        )}
      >
        <Link
          to={to}
          aria-current={selected ? "page" : undefined}
          className="min-w-0 flex-1 py-2 outline-none after:absolute after:inset-0 focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
        >
          <span className="flex min-w-0 items-start gap-1.5 text-[13px]">
            {leading && <span className="flex h-lh shrink-0 items-center">{leading}</span>}
            <span dir="auto" title={title} className="line-clamp-2 min-w-0 font-medium break-words text-foreground">
              {title}
            </span>
            {titleIcon && <span className="flex h-lh shrink-0 items-center">{titleIcon}</span>}
          </span>
          {snippet !== "" && (
            <span dir="auto" className="block truncate text-xs text-muted-foreground">
              {snippet}
            </span>
          )}
        </Link>
        {!!meta && (
          <div
            className={cn(
              "flex shrink-0 items-center gap-1.5",
              hasActions && "group-focus-within:hidden group-hover:hidden group-has-[[data-state=open]]:hidden",
              hasActions && selected && "hidden",
            )}
          >
            {meta}
          </div>
        )}
        {hasActions && (
          <div
            className={cn(
              "relative z-10 shrink-0 items-center gap-0.5 group-focus-within:flex group-hover:flex has-[[data-state=open]]:flex",
              selected ? "flex" : "hidden",
            )}
          >
            {actions}
            {offersMenu(menu) && <RowActions itemLabel={title} {...menu} />}
          </div>
        )}
        {!!trailing && <div className="relative z-10 flex shrink-0 items-center">{trailing}</div>}
      </li>
    </RowContextMenu>
  );
};
