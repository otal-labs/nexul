import { CopyButton } from "@/components/settings/CopyButton";

interface CopyValueProps {
  value: string;
  // Names what is copied, for the icon button.
  label: string;
}

// A value to copy, in a mono well the height and type size of a field, so it lines up with the inputs beside it.
export const CopyValue = ({ value, label }: CopyValueProps) => (
  <span className="flex h-9 w-full min-w-0 items-center gap-1 rounded-md bg-surface-2 pr-0.5 pl-3 ring-1 ring-border">
    <code className="min-w-0 flex-1 truncate font-mono text-sm" title={value}>
      {value}
    </code>
    <CopyButton value={value} label={label} iconOnly variant="ghost" />
  </span>
);
