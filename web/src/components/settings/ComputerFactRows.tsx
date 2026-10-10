import { ComputerFact } from "@/components/settings/ComputerFact";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { formatBytes } from "@/models/Attachment";
import { SIGN_IN, type ComputerFacts, type ProviderFacts } from "@/models/ComputerFacts";
import { formatRelativeTime } from "@/utils/TimeUtility";

// Past this many, a computer's T3 Code projects end in a count, so the fold stays a glance.
const SHOWN_PROJECTS = 5;

interface ProviderFactLineProps {
  provider: ProviderFacts;
}

const ProviderFactLine = ({ provider }: ProviderFactLineProps) => {
  const signIn = SIGN_IN[provider.sign_in];
  const models = `${provider.models.length} ${provider.models.length === 1 ? "model" : "models"}`;
  return (
    <li className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
      <span className="break-words text-foreground">{provider.name}</span>
      <SettingsStatus tone={signIn.tone} detail={<span className="font-mono tabular-nums">{[provider.version, models].filter(Boolean).join(" · ")}</span>}>
        {signIn.text}
      </SettingsStatus>
    </li>
  );
};

interface ProjectFactLineProps {
  project: ComputerFacts["projects"][number];
}

const ProjectFactLine = ({ project }: ProjectFactLineProps) => (
  <li className="flex min-w-0 flex-wrap gap-x-2">
    <span className="text-foreground">{project.title}</span>
    <span className="min-w-0 font-mono break-all text-muted-foreground">{project.path}</span>
  </li>
);

interface ComputerFactRowsProps {
  facts: ComputerFacts;
  factsAt: string | undefined;
}

// What the computer's runner and its T3 Code report; only the computer's owner ever receives them.
export const ComputerFactRows = ({ facts, factsAt }: ComputerFactRowsProps) => {
  const system = [facts.os, facts.arch].filter(Boolean).join(" · ");
  const git = [facts.git_name, facts.git_email && `<${facts.git_email}>`].filter(Boolean).join(" ");
  const hidden = facts.projects.length - SHOWN_PROJECTS;
  return (
    <>
      {facts.providers.length > 0 && (
        <ComputerFact label="Provider CLIs">
          <ul className="space-y-1">
            {facts.providers.map((provider) => (
              <ProviderFactLine key={provider.id} provider={provider} />
            ))}
          </ul>
        </ComputerFact>
      )}
      {facts.projects.length > 0 && (
        <ComputerFact label="T3 projects">
          <ul className="space-y-1">
            {facts.projects.slice(0, SHOWN_PROJECTS).map((project) => (
              <ProjectFactLine key={project.id} project={project} />
            ))}
            {hidden > 0 && <li className="text-muted-foreground">and {hidden} more</li>}
          </ul>
        </ComputerFact>
      )}
      {(facts.hostname || system) && (
        <ComputerFact label="Computer">
          <span className="font-mono break-all">{facts.hostname}</span>
          {system && <span className="ml-2 font-mono text-muted-foreground">{system}</span>}
        </ComputerFact>
      )}
      {git && (
        <ComputerFact label="Git identity">
          <span className="break-all">{git}</span>
        </ComputerFact>
      )}
      {facts.free_disk_bytes !== undefined && (
        <ComputerFact label="Free disk">
          <span className="font-mono tabular-nums">{formatBytes(facts.free_disk_bytes)}</span>
        </ComputerFact>
      )}
      {facts.runner_version && (
        <ComputerFact label="Nexul app">
          <span className="font-mono">{facts.runner_version}</span>
        </ComputerFact>
      )}
      {facts.cloudflared && (
        <ComputerFact label="cloudflared">
          <span className="font-mono">{facts.cloudflared}</span>
        </ComputerFact>
      )}
      {factsAt && (
        <ComputerFact label="Facts changed">
          <span className="font-mono tabular-nums">{formatRelativeTime(factsAt)}</span>
        </ComputerFact>
      )}
    </>
  );
};
