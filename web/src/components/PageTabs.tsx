import { useRef, type ReactNode } from "react";
import { useLocation, useNavigate } from "react-router";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { TabUnderline } from "@/components/ActiveIndicator";
import { UpdateDot } from "@/components/UpdateDot";
import { useSwapEntrance } from "@/hooks/useSwapEntrance";
import { useTabPath } from "@/hooks/useTabPath";
import { cn } from "@/lib/utils";

export interface PageTab {
  value: string;
  label: string;
  /** A hidden tab has no trigger and can't be selected, even by its path segment. */
  hidden?: boolean;
  /** Set to show the update dot beside the label, with this as its screen-reader hint. */
  dot?: string | undefined;
}

interface PageTabsProps {
  label: string;
  tabs: PageTab[];
  children: ReactNode;
  className?: string;
}

// The active tab is the path's last segment so it survives a reload and can be linked; the first visible tab has no segment, and a missing or unknown one falls back to it.
export const PageTabs = ({ label, tabs, children, className }: PageTabsProps) => {
  const navigate = useNavigate();
  const { search } = useLocation();
  const { current, tabPath } = useTabPath();
  const root = useRef<HTMLDivElement>(null);
  const visible = tabs.filter((tab) => !tab.hidden);
  const active = visible.find((tab) => tab.value === current) ?? visible[0];
  useSwapEntrance(
    active?.value ?? "",
    () => (active ? visible.indexOf(active) : 0),
    () => root.current?.querySelector(':scope > [role="tabpanel"][data-state="active"]'),
    () => "x",
  );
  if (!active) return null;

  const select = (value: string) =>
    navigate({ pathname: tabPath(value === visible[0]?.value ? undefined : value), search });

  return (
    <Tabs ref={root} value={active.value} onValueChange={select} className={cn("gap-6", className)}>
      {visible.length > 1 && (
        <TabsList
          variant="line"
          aria-label={label}
          className="relative isolate w-full justify-start overflow-x-auto border-b border-border p-0"
        >
          <TabUnderline />
          {visible.map((tab) => (
            <TabsTrigger
              key={tab.value}
              value={tab.value}
              className="flex-none px-3 after:hidden"
            >
              <span className="relative">
                {tab.label}
                {tab.dot && <UpdateDot label={tab.dot} className="-top-0.5 -right-2 ring-0" />}
              </span>
            </TabsTrigger>
          ))}
        </TabsList>
      )}
      {children}
    </Tabs>
  );
};

interface PageTabsContentProps {
  value: string;
  children: ReactNode;
  className?: string;
}

export const PageTabsContent = ({ value, children, className }: PageTabsContentProps) => (
  <TabsContent value={value} className={cn("min-w-0 space-y-6", className)}>
    {children}
  </TabsContent>
);
