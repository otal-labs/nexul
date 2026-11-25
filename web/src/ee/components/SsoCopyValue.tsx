import { CopyButton } from "@/components/settings/CopyButton";

interface SsoCopyValueProps {
  value: string;
  // Names what is copied, for the icon button.
  label: string;
}

// A value the provider registers, in a mono well with its copy button.
export const SsoCopyValue = ({ value, label }: SsoCopyValueProps) => (
  <span className="flex w-full min-w-0 items-center gap-1 rounded-md bg-surface-2 py-0.5 pr-0.5 pl-2.5 ring-1 ring-border">
    <code className="min-w-0 flex-1 truncate font-mono text-xs" title={value}>
      {value}
    </code>
    <CopyButton value={value} label={label} iconOnly variant="ghost" />
  </span>
);
