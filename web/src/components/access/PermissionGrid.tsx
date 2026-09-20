import { Checkbox } from "@/components/ui/checkbox";
import type { PermissionInfo } from "@/models/Permission";

interface PermissionGridProps {
  entries: PermissionInfo[];
  value: string[];
  onChange: (value: string[]) => void;
}

// Columns are whatever actions the catalog actually carries, in first-seen order (read, write, delete, then verbs).
const columnsOf = (entries: PermissionInfo[]): string[] => {
  const columns: string[] = [];
  for (const entry of entries) {
    if (!columns.includes(entry.action)) columns.push(entry.action);
  }
  return columns;
};

// Server sends entries pre-grouped by domain; this only buckets them, it never reorders.
const groupByDomain = (entries: PermissionInfo[]): [string, PermissionInfo[]][] => {
  const order: string[] = [];
  const byDomain = new Map<string, PermissionInfo[]>();
  for (const entry of entries) {
    const bucket = byDomain.get(entry.domain);
    if (bucket) {
      bucket.push(entry);
      continue;
    }
    byDomain.set(entry.domain, [entry]);
    order.push(entry.domain);
  }
  return order.map((domain) => [domain, byDomain.get(domain) ?? []]);
};

// No client-side domain name map: the read entry's own label ("Read docs") minus its verb is the row name.
const domainLabel = (entries: PermissionInfo[]): string => {
  const [first] = entries;
  const source = entries.find((entry) => entry.action === "read") ?? first;
  if (!source) return "";
  const text = source.action === "read" ? source.label.replace(/^Read\s+/, "") : source.label;
  return text.charAt(0).toUpperCase() + text.slice(1);
};

const capitalize = (text: string): string => text.charAt(0).toUpperCase() + text.slice(1);

interface PermissionGridRowProps {
  entries: PermissionInfo[];
  columns: string[];
  value: string[];
  onChange: (value: string[]) => void;
}

const PermissionGridRow = ({ entries, columns, value, onChange }: PermissionGridRowProps) => {
  const toggle = (entryValue: string, checked: boolean) =>
    onChange(checked ? [...value, entryValue] : value.filter((v) => v !== entryValue));

  return (
    <>
      <span className="truncate text-sm">{domainLabel(entries)}</span>
      {columns.map((action) => {
        const entry = entries.find((e) => e.action === action);
        return (
          <span key={action} className="flex justify-center">
            {entry && (
              <Checkbox
                aria-label={entry.label}
                checked={value.includes(entry.value)}
                onCheckedChange={(next) => toggle(entry.value, !!next)}
              />
            )}
          </span>
        );
      })}
    </>
  );
};

// One row per domain, one column per action the catalog carries; a cell only renders where a domain offers it.
export const PermissionGrid = ({ entries, value, onChange }: PermissionGridProps) => {
  const columns = columnsOf(entries);
  const gridTemplateColumns = `minmax(0,1fr) repeat(${columns.length}, 2.75rem)`;

  return (
    <div
      className="grid items-center gap-x-2 gap-y-1.5 rounded-md border border-input p-2"
      style={{ gridTemplateColumns }}
    >
      <span />
      {columns.map((action) => (
        <span key={action} className="text-center text-[11px] font-medium text-muted-foreground">
          {capitalize(action)}
        </span>
      ))}
      {groupByDomain(entries).map(([domain, domainEntries]) => (
        <PermissionGridRow key={domain} entries={domainEntries} columns={columns} value={value} onChange={onChange} />
      ))}
    </div>
  );
};
