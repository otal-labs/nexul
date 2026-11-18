import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { useStartImport } from "@/hooks/ImportHooks";
import { useFetchProjectRepos, useFetchProjects } from "@/hooks/ProjectHooks";
import { IMPORT_SOURCES } from "@/models/Import";

// Picks what to import and where; Scan only counts, so nothing is written until the scan is confirmed.
export const StartImportCard = () => {
  const [source, setSource] = useState(IMPORT_SOURCES[0]!.id);
  const [projectId, setProjectId] = useState("");
  const [picked, setPicked] = useState("");
  const { data: projects } = useFetchProjects();
  const { data: repos } = useFetchProjectRepos(projectId || undefined);
  const start = useStartImport();
  const sourceHint = IMPORT_SOURCES.find((s) => s.id === source)?.hint;
  // A project with one repository imports it; with several, the person names one.
  const from = repos?.length === 1 ? repos[0]!.full_name : picked;
  const ready = projectId !== "" && !!repos && repos.length > 0 && from !== "";

  return (
    <SettingsCard
      id="import-start"
      title="Import into a project"
      description="Brings another tool's records in as tickets. A scan counts them and shows how people and statuses map before anything is written; a re-run skips what came in before."
      footer={
        <Button
          className="ml-auto"
          disabled={!ready}
          loading={start.isPending}
          onClick={() => start.mutate({ source, project_id: projectId, from })}
        >
          Scan
        </Button>
      }
    >
      <SettingsRows>
        <SettingsRow label="Source" htmlFor="import-source" description={sourceHint}>
          <Select value={source} onValueChange={setSource}>
            <SelectTrigger id="import-source" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {IMPORT_SOURCES.map((s) => (
                <SelectItem key={s.id} value={s.id}>
                  {s.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </SettingsRow>
        <SettingsRow
          label="Project"
          htmlFor="import-project"
          description="Tickets land in its columns; a status it lacks becomes a new column."
        >
          <Select
            value={projectId}
            onValueChange={(id) => {
              setProjectId(id);
              setPicked("");
            }}
          >
            <SelectTrigger id="import-project" className="w-full">
              <SelectValue placeholder="Pick a project" />
            </SelectTrigger>
            <SelectContent>
              {projects?.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </SettingsRow>
        {repos && repos.length === 0 && (
          <SettingsRow label="Repository" description="This project has no repository. Attach one in its settings first." />
        )}
        {repos && repos.length === 1 && (
          <SettingsRow label="Repository">
            <span className="font-mono text-sm">{repos[0]!.full_name}</span>
          </SettingsRow>
        )}
        {repos && repos.length > 1 && (
          <SettingsRow label="Repository" htmlFor="import-repository" description="The project has several; issues come from one.">
            <Select value={picked} onValueChange={setPicked}>
              <SelectTrigger id="import-repository" className="w-full">
                <SelectValue placeholder="Pick a repository" />
              </SelectTrigger>
              <SelectContent>
                {repos.map((r) => (
                  <SelectItem key={r.full_name} value={r.full_name} className="font-mono">
                    {r.full_name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </SettingsRow>
        )}
      </SettingsRows>
    </SettingsCard>
  );
};
