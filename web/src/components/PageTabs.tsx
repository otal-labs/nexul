import type { ReactNode } from "react";
import { useSearchParams } from "react-router";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";

export interface PageTab {
  value: string;
  label: string;
  /** A hidden tab has no trigger and can't be selected, even by `?tab=`. */
  hidden?: boolean;
}

interface PageTabsProps {
  label: string;
  tabs: PageTab[];
  children: ReactNode;
  className?: string;
}

// The active tab lives in `?tab=` so it survives a reload and can be linked; a missing or unknown value falls back to the first visible tab.
export const PageTabs = ({ label, tabs, children, className }: PageTabsProps) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const visible = tabs.filter((tab) => !tab.hidden);
  const requested = searchParams.get("tab");
  const active = visible.find((tab) => tab.value === requested) ?? visible[0];
  if (!active) return null;

  const select = (value: string) =>
    setSearchParams((params) => {
      params.set("tab", value);
      return params;
    });

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
