import { cn } from "@/lib/utils";

export type AutomationTab = "overview" | "versions";

const TABS: { id: AutomationTab; label: string }[] = [
  { id: "overview", label: "Overview" },
  { id: "versions", label: "Versions" },
];

interface AutomationTabsNavProps {
  active: AutomationTab;
  onSelect: (tab: AutomationTab) => void;
}

export const AutomationTabsNav = ({ active, onSelect }: AutomationTabsNavProps) => (
  <div role="tablist" aria-label="Automation sections" className="flex w-fit gap-1 rounded-md border p-1">
    {TABS.map((t) => (
      <button
        key={t.id}
        type="button"
        role="tab"
        aria-selected={active === t.id}
        onClick={() => onSelect(t.id)}
        className={cn(
          "rounded px-3 py-1 text-sm font-medium transition-colors duration-150 ease-standard",
          active === t.id ? "bg-accent text-primary" : "text-muted-foreground hover:text-foreground",
        )}
      >
        {t.label}
      </button>
    ))}
  </div>
);
