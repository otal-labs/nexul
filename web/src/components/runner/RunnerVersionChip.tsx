import { useServerVersion } from "@/hooks/VersionHooks";

interface RunnerVersionChipProps {
  version: string;
}

// "dev" (local builds) never counts as behind — there is no release tag to compare it to. Runners
// update themselves on their next connect, so this only ever informs, it never links to an action.
export const RunnerVersionChip = ({ version }: RunnerVersionChipProps) => {
  const server = useServerVersion();

  return (
    <span className="inline-flex flex-wrap items-center gap-2">
      <span className="font-mono text-xs text-muted-foreground">{version}</span>
      {server.data && version !== "dev" && server.data.version !== "dev" && version !== server.data.version && (
        <span
          className="inline-flex shrink-0 items-center gap-1.5 font-mono text-xs text-muted-foreground"
          title="Runners update themselves on their next connect"
        >
          <span aria-hidden className="size-1.5 rounded-full bg-warning" />
          updating · {server.data.version}
        </span>
      )}
    </span>
  );
};
