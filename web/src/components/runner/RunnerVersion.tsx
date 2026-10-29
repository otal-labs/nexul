import { useServerVersion } from "@/hooks/VersionHooks";

interface RunnerVersionProps {
  version: string;
}

// "dev" (local builds) never counts as behind; a behind runner updates itself on its next connect, so this only informs.
export const RunnerVersion = ({ version }: RunnerVersionProps) => {
  const server = useServerVersion();
  const target = server.data?.version;
  const behind = !!target && version !== "" && version !== "dev" && target !== "dev" && version !== target;
  return (
    <span
      className="truncate font-mono text-xs text-muted-foreground tabular-nums"
      title={behind ? `Updates to ${target} on its next connect` : undefined}
    >
      {version}
      {behind && <span aria-hidden> ↑</span>}
      {behind && <span className="sr-only">, updates to {target}</span>}
    </span>
  );
};
