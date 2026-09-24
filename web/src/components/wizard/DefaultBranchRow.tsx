interface DefaultBranchRowProps {
  branch: string;
  hostname: string | undefined;
  network: string;
}

export const DefaultBranchRow = ({ branch, hostname, network }: DefaultBranchRowProps) => (
  <li className="space-y-1 py-4">
    <div className="flex items-baseline justify-between gap-3">
      <p className="font-mono text-sm">{branch}</p>
      <p className="text-xs text-muted-foreground">Production</p>
    </div>
    <p className="text-sm text-muted-foreground">Every push redeploys this service in place.</p>
    <p className="font-mono text-xs wrap-break-word text-muted-foreground">
      {branch} → {hostname ?? "no hostname"} on {network}
    </p>
  </li>
);
