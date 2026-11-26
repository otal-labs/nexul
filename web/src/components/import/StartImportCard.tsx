import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { ImportTextRow } from "@/components/import/ImportTextRow";
import { LinearImportRows } from "@/components/import/LinearImportRows";
import { NotionImportRows } from "@/components/import/NotionImportRows";
import { useStartImport } from "@/hooks/ImportHooks";
import { useFetchProjectRepos, useFetchProjects } from "@/hooks/ProjectHooks";
import { IMPORT_SOURCES } from "@/models/Import";

// Picks what to import and where; Scan only counts, so nothing is written until the scan is confirmed.
export const StartImportCard = () => {
  const [source, setSource] = useState(IMPORT_SOURCES[0]!.id);
  const [projectId, setProjectId] = useState("");
  const [picked, setPicked] = useState("");
  const [token, setToken] = useState("");
  const [pages, setPages] = useState("");
  const [team, setTeam] = useState("");
  const [link, setLink] = useState("");
  const { data: projects } = useFetchProjects();
  const sourceInfo = IMPORT_SOURCES.find((s) => s.id === source);
  const asks = sourceInfo?.asks;
  const asksRepository = asks === "repository";
  const { data: repos } = useFetchProjectRepos(asksRepository && projectId ? projectId : undefined);
  const start = useStartImport();
  const sourceHint = sourceInfo?.hint;
  // A project with one repository imports it; with several, the person names one.
  const repository = repos?.length === 1 ? repos[0]!.full_name : picked;
  const ready = asksRepository
    ? projectId !== "" && !!repos && repos.length > 0 && repository !== ""
    : projectId !== "" && token.trim() !== "";
  // What still holds Scan back, said under it one step at a time.
  const needs = [
    projectId === "" && "Pick a project",
    asksRepository && repos?.length === 0 && "Attach a repository to the project",
    asksRepository && repos && repos.length > 1 && repository === "" && "Pick a repository",
    !asksRepository && token.trim() === "" && `Add the ${(sourceInfo?.credential?.label ?? "token").toLowerCase()}`,
  ].find(Boolean);
  const input = asksRepository
    ? { source, project_id: projectId, from: repository }
    : { source, project_id: projectId, from: { team, pages, link, repository: "" }[asks ?? "repository"].trim(), credential: token.trim() };

  return (
    <SettingsCard
      id="import-start"
      title="Import into a project"
      description="Brings another tool's records in as tickets or docs. A scan counts them and shows how people and statuses map before anything is written; a re-run skips what came in before."
      footer={
        <>
          {needs && <SettingsHint>{needs} to scan.</SettingsHint>}
          <Button className="ml-auto" disabled={!ready} loading={start.isPending} onClick={() => start.mutate(input)}>
            Scan
          </Button>
        </>
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
          description={
            asks === "pages"
              ? "Docs land in a folder per root page, made if the project lacks one."
              : "Tickets land in its columns; a status it lacks becomes a new column."
          }
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
        {asks === "pages" && <NotionImportRows token={token} pages={pages} onToken={setToken} onPages={setPages} />}
        {asks === "team" && <LinearImportRows token={token} team={team} onToken={setToken} onTeam={setTeam} />}
        {sourceInfo?.from && <ImportTextRow id="import-from" field={sourceInfo.from} value={link} onChange={setLink} />}
        {sourceInfo?.credential && (
          <ImportTextRow id="import-credential" field={sourceInfo.credential} value={token} onChange={setToken} secret />
        )}
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
