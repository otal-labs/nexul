import { Link } from "react-router";

import type { VersionChange, VersionLatest } from "@/models/Version";

export const INSTANCE_VERSION_SECTION_URL = "/settings/instance#instance-version";

interface UpdateChangelogProps {
  current: string;
  latest: VersionLatest;
  changes: VersionChange[];
}

const UpdateChangeItem = ({ change }: { change: VersionChange }) => (
  <li>
    <a
      href={change.url}
      target="_blank"
      rel="noreferrer"
      className="font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
    >
      {change.version}
    </a>
    {change.notes.length > 0 && (
      <ul className="mt-1 list-disc space-y-1 pl-4 marker:text-muted-foreground">
        {change.notes.map((note) => (
          <li key={note} className="text-sm leading-snug">
            {note}
          </li>
        ))}
      </ul>
    )}
  </li>
);

export const UpdateChangelog = ({ current, latest, changes }: UpdateChangelogProps) => (
  <div className="flex flex-col">
    <div className="border-b border-border px-3 py-2.5">
      <p className="text-sm font-medium">Update available</p>
      <p className="font-mono text-xs text-muted-foreground">
        {current} → {latest.version}
      </p>
    </div>
    {changes.length > 0 && (
      <ul className="max-h-72 space-y-3 overflow-y-auto px-3 py-2.5">
        {changes.map((change) => (
          <UpdateChangeItem key={change.version} change={change} />
        ))}
      </ul>
    )}
    {changes.length === 0 && (
      <a
        href={latest.url}
        target="_blank"
        rel="noreferrer"
        className="px-3 py-2.5 text-sm text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
      >
        Release notes
      </a>
    )}
    <Link
      to={INSTANCE_VERSION_SECTION_URL}
      className="border-t border-border px-3 py-2 text-sm font-medium transition-colors duration-150 ease-standard hover:bg-accent/40"
    >
      Upgrade in Settings
    </Link>
  </div>
);
