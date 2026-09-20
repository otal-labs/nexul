import { AlertCircle, ExternalLink, SearchIcon } from "lucide-react";
import { useMemo, useState } from "react";

import { errorMessage } from "@/api/client";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useFetchRepositories, useScanRepository } from "@/hooks/RepositoryHooks";
import { manualCandidate, parseInstallUrl, type Repo } from "@/models/Repository";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

interface RepositoryRowProps {
  repo: Repo;
  busy: boolean;
  onSelect: () => void;
}

const RepositoryRow = ({ repo, busy, onSelect }: RepositoryRowProps) => (
  <li>
    <button
      type="button"
      onClick={onSelect}
      disabled={busy}
      className="flex w-full items-center justify-between gap-3 px-2 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent/40 disabled:opacity-60"
    >
      <span className="min-w-0">
        <span className="block truncate text-sm font-medium">{repo.full_name}</span>
        <span className="block truncate font-mono text-xs text-muted-foreground">{repo.default_branch}</span>
      </span>
      {busy && <span className="shrink-0 text-xs text-muted-foreground">Scanning…</span>}
    </button>
  </li>
);

interface WizardRepositoryStepProps {
  onDone: () => void;
}

// Lists installation repositories (spec §5/§7); picking one scans it. A scan can land in three places: candidates
// found (advance), nothing found (offer a manual Dockerfile candidate), or the App isn't installed (link to fix it).
export const WizardRepositoryStep = ({ onDone }: WizardRepositoryStepProps) => {
  const { data: repos, isPending, error } = useFetchRepositories();
  const scanRepository = useScanRepository();
  const setRepository = useProjectWizardStore((s) => s.setRepository);
  const setScanResult = useProjectWizardStore((s) => s.setScanResult);
  const setCandidate = useProjectWizardStore((s) => s.setCandidate);
  const [query, setQuery] = useState("");
  const [picked, setPicked] = useState<Repo | null>(null);
  const [dockerfilePath, setDockerfilePath] = useState("Dockerfile");

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!repos) return [];
    if (!q) return repos;
    return repos.filter((r) => r.full_name.toLowerCase().includes(q));
  }, [repos, query]);

  const scan = async (repo: Repo) => {
    setPicked(repo);
    try {
      const result = await scanRepository.mutateAsync({ owner: repo.owner, name: repo.name });
      setRepository(repo);
      setScanResult(result);
      if (result.candidates.length > 0) {
        const preferred = result.candidates.find((c) => c.kind === "compose") ?? result.candidates[0]!;
        setCandidate(preferred);
        onDone();
      }
    } catch {
      // Rendered inline below (not-installed vs generic failure vs empty candidates); no toast for this one.
    }
  };

  const continueManually = () => {
    if (!picked || !dockerfilePath.trim()) return;
    setRepository(picked);
    setCandidate(manualCandidate(picked.name, dockerfilePath.trim()));
    onDone();
  };

  const notInstalled =
    scanRepository.isError &&
    (scanRepository.error as { response?: { status?: number } })?.response?.status === 404;
  const installUrl = notInstalled ? parseInstallUrl(errorMessage(scanRepository.error)) : undefined;
  const nothingFound = scanRepository.isSuccess && scanRepository.data.candidates.length === 0;

  return (
    <div className="space-y-4">
      <div className="relative">
        <SearchIcon
          className="absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden
        />
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search repositories…"
          aria-label="Search repositories"
          className="pl-8"
        />
      </div>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Could not load repositories" />}
      {repos && filtered.length === 0 && (
        <EmptyState title="No repositories found" message="Try a different search." size="compact" />
      )}
      {repos && filtered.length > 0 && (
        <ul className="divide-y divide-border border-y border-border">
          {filtered.map((repo) => (
            <RepositoryRow
              key={repo.id}
              repo={repo}
              busy={scanRepository.isPending && picked?.id === repo.id}
              onSelect={() => void scan(repo)}
            />
          ))}
        </ul>
      )}
      {notInstalled && (
        <EmptyState
          icon={AlertCircle}
          title="Not installed on this repository"
          message={errorMessage(scanRepository.error)}
          size="compact"
          action={
            installUrl && (
              <Button asChild variant="outline" size="sm">
                <a href={installUrl} target="_blank" rel="noreferrer">
                  Install the GitHub App
                  <ExternalLink className="size-3.5" aria-hidden />
                </a>
              </Button>
            )
          }
        />
      )}
      {scanRepository.isError && !notInstalled && <ErrorDisplay error={scanRepository.error} title="Scan failed" />}
      {nothingFound && (
        <EmptyState
          title="Nothing to deploy found"
          message="No Dockerfile or compose file in this repository. Point the wizard at a Dockerfile yourself to continue."
          size="compact"
          action={
            <div className="flex flex-wrap items-center justify-center gap-2">
              <Input
                value={dockerfilePath}
                onChange={(e) => setDockerfilePath(e.target.value)}
                placeholder="Dockerfile"
                aria-label="Dockerfile path"
                className="h-8 w-40"
              />
              <Button size="sm" onClick={continueManually} disabled={!dockerfilePath.trim()}>
                Continue manually
              </Button>
            </div>
          }
        />
      )}
    </div>
  );
};
