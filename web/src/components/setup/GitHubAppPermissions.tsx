import { microheaderClass } from "@/components/Microheader";

const PERMISSIONS = [
  { name: "Contents", access: "read", why: "to clone and build" },
  { name: "Pull requests", access: "read and write", why: "to link them to tickets" },
  { name: "Webhooks", access: "read and write", why: "to hear about pushes" },
];

// What to tick on GitHub's form, one row per repository permission; Metadata is always included.
export const GitHubAppPermissions = () => (
  <div>
    <p className={microheaderClass}>Repository permissions</p>
    <ul className="mt-1.5 divide-y divide-border border-y border-border">
      {PERMISSIONS.map((p) => (
        <li key={p.name} className="flex items-baseline justify-between gap-4 py-2 text-[13px]">
          <span>{p.name}<span className="text-muted-foreground"> · {p.why}</span></span>
          <span className="font-mono text-xs text-muted-foreground">{p.access}</span>
        </li>
      ))}
    </ul>
  </div>
);
