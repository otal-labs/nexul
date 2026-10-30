import { useEffect, useRef, type ReactNode } from "react";
import { Link } from "react-router";

import { currentRowFillClass, CurrentRowEdge } from "@/components/CurrentRow";
import type { RowMenu } from "@/components/listpane/rowMenu";
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
  /** The row's right-click menu. */
  menu?: RowMenu | undefined;
  /** Always visible at the right edge, e.g. a switch. */
  trailing?: ReactNode;
  /** One truncated line at a 32px pitch instead of up to two wrapped lines. */
  dense?: boolean;
  /** Sets meta and trailing on the title's first line instead of centring them on a row whose title may wrap. */
  metaOnTitleLine?: boolean;
}

// The link's ::after covers the row so the whole row opens it; actions and trailing sit above that layer.
export const ListPaneRow = ({ to, title, leading, titleIcon, snippet = "", selected, meta, actions, menu, trailing, dense = false, metaOnTitleLine = false }: ListPaneRowProps) => {
  const onTitleLine = metaOnTitleLine && "mt-2 h-lh text-[13px]";
  const ref = useRef<HTMLLIElement>(null);
  const hasActions = !!actions;

  // A deep link can select a row far down the list.
  useEffect(() => {
    if (selected) ref.current?.scrollIntoView?.({ block: "nearest" });
  }, [selected]);

  return (
    <RowContextMenu menu={menu}>
      <li
        ref={ref}
        className={cn(
          "group relative flex gap-2 px-3 transition-colors duration-150 ease-standard",
          metaOnTitleLine ? "items-start" : "items-center",
          snippet === "" && (dense ? "min-h-8" : "min-h-10"),
          snippet !== "" && "min-h-14",
          selected ? currentRowFillClass : "hover:bg-accent",
        )}
      >
        {selected && <CurrentRowEdge />}
        <Link
          to={to}
          aria-current={selected ? "page" : undefined}
          className={cn("quiet-focus min-w-0 flex-1 after:absolute after:inset-0 after:rounded-sm focus-visible:after:outline-2 focus-visible:after:-outline-offset-2 focus-visible:after:outline-focus", dense ? "py-1.5" : "py-2")}
        >
          <span className="flex min-w-0 items-start gap-1.5 text-[13px]">
            {leading && <span className="flex h-lh shrink-0 items-center">{leading}</span>}
            <span dir="auto" title={title} className={cn("min-w-0 text-foreground", dense ? "truncate" : "line-clamp-2 break-words text-balance", selected && "font-medium")}>
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
              onTitleLine,
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
          </div>
        )}
        {!!trailing && <div className={cn("relative z-10 flex shrink-0 items-center", onTitleLine)}>{trailing}</div>}
      </li>
    </RowContextMenu>
  );
};
