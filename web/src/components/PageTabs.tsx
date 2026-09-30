import type { ReactNode } from "react";
import { useLocation, useNavigate } from "react-router";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useTabPath } from "@/hooks/useTabPath";
import { cn } from "@/lib/utils";

export interface PageTab {
  value: string;
  label: string;
  /** A hidden tab has no trigger and can't be selected, even by its path segment. */
  hidden?: boolean;
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
  const visible = tabs.filter((tab) => !tab.hidden);
  const active = visible.find((tab) => tab.value === current) ?? visible[0];
  if (!active) return null;

  const select = (value: string) =>
    navigate({ pathname: tabPath(value === visible[0]?.value ? undefined : value), search });

  return (
    <Tabs value={active.value} onValueChange={select} className={cn("gap-6", className)}>
      {visible.length > 1 && (
        <TabsList
          variant="line"
          aria-label={label}
          className="w-full justify-start overflow-x-auto border-b border-border p-0"
        >
          {visible.map((tab) => (
            <TabsTrigger
              key={tab.value}
              value={tab.value}
              className="flex-none px-3 group-data-[orientation=horizontal]/tabs:after:bottom-[-1px]"
            >
              {tab.label}
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
