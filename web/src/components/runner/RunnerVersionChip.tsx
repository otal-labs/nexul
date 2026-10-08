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
          className="inline-flex shrink-0 items-center rounded-full bg-warning/15 px-2 py-0.5 font-mono text-[11px] font-medium text-warning"
          title="Runners update themselves on their next connect"
        >
          updating · {server.data.version}
        </span>
      )}
    </span>
  );
};
