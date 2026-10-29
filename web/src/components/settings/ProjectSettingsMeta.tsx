import { HUE_DOT_CLASS } from "@/components/board/ticketTypeColor";
import type { ProjectSettingsSection } from "@/components/settings/ProjectSettingsNav";
import { useFetchProjectCategories } from "@/hooks/CategoryHooks";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchGrants } from "@/hooks/PermissionHooks";
import { useFetchProjectDeleteImpact, useFetchProjectRepos } from "@/hooks/ProjectHooks";
import { useFetchServices } from "@/hooks/ServiceHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchAllLabels, useFetchLabelColors } from "@/hooks/TicketHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import type { Project } from "@/models/Project";
import { projectPeopleRules } from "@/models/ProjectPeople";
import { RepoRole } from "@/enums/Project";

interface ProjectSettingsMetaProps {
  project: Project;
  section: ProjectSettingsSection;
}

const count = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;

const Facts = ({ parts }: { parts: (string | false | undefined)[] }) => (
  <span className="font-mono text-xs tabular-nums">{parts.filter(Boolean).join(" · ")}</span>
);

// The project's size; services as the Services section lists them, without branch deploys.
const SizeMeta = ({ project }: { project: Project }) => {
  const { data: impact } = useFetchProjectDeleteImpact(project.id);
  const canReadStacks = useAreaAccess(project.id)?.("stacks") ?? false;
  const { data: services } = useFetchServices(project.id, canReadStacks);
  const serviceCount = canReadStacks ? services?.length : impact?.services;
  const empty = impact && impact.tickets + impact.repos + impact.services === 0;
  return (
    <Facts
      parts={[
        project.prefix,
        empty && "empty",
        impact && !empty && count(impact.tickets, "ticket"),
        impact && !empty && count(impact.repos, "repo"),
        impact && !empty && serviceCount !== undefined && count(serviceCount, "service"),
      ]}
    />
  );
};

const AccessMeta = ({ projectId }: { projectId: string }) => {
  const { data: grants } = useFetchGrants("project", projectId);
  const rules = grants && projectPeopleRules(grants);
  const open = rules && rules.roles.length + rules.people.length === 0;
  return <Facts parts={[open && "Owners only", rules && !open && count(rules.roles.length, "role"), rules && !open && count(rules.people.length, "person", "people"), !!rules?.keptOut.length && `${rules.keptOut.length} kept out`]} />;
};

const StatusesMeta = ({ projectId }: { projectId: string }) => {
  const { data: statuses } = useFetchProjectStatuses(projectId);
  const stages = statuses && new Set(statuses.map((s) => s.kind)).size;
  return <Facts parts={[statuses && count(statuses.length, "status", "statuses"), stages !== undefined && count(stages, "stage")]} />;
};

const CategoriesMeta = ({ projectId }: { projectId: string }) => {
  const { data: categories } = useFetchProjectCategories(projectId);
  return <Facts parts={[categories && count(categories.length, "category", "categories")]} />;
};

const TypesMeta = ({ projectId }: { projectId: string }) => {
  const { data: types } = useFetchProjectTicketTypes(projectId);
  return <Facts parts={[types && count(types.length, "type")]} />;
};

const LabelsMeta = ({ projectId }: { projectId: string }) => {
  const { data: labels } = useFetchAllLabels(projectId);
  const { data: colors } = useFetchLabelColors(projectId);
  const unset = labels && colors && labels.filter((label) => !HUE_DOT_CLASS[colors[label] ?? ""]).length;
  return <Facts parts={[labels && count(labels.length, "label"), !!unset && `${unset} without a color`]} />;
};

const ReposMeta = ({ projectId }: { projectId: string }) => {
  const { data: repos } = useFetchProjectRepos(projectId);
  const tests = repos?.filter((r) => r.role === RepoRole.Tests).length;
  return <Facts parts={[repos && count(repos.length, "repo"), !!tests && `${tests} for tests`]} />;
};

const ServicesMeta = ({ projectId }: { projectId: string }) => {
  const { data: services } = useFetchServices(projectId);
  return <Facts parts={[services && count(services.length, "service")]} />;
};

// The header's facts line for the open section: what it holds right now, from the query the section already loads.
export const ProjectSettingsMeta = ({ project, section }: ProjectSettingsMetaProps) => (
  <>
    {section === "general" && <SizeMeta project={project} />}
    {section === "danger" && <Facts parts={[project.prefix, "removing a project can't be undone"]} />}
    {section === "access" && <AccessMeta projectId={project.id} />}
    {section === "statuses" && <StatusesMeta projectId={project.id} />}
    {section === "categories" && <CategoriesMeta projectId={project.id} />}
    {section === "types" && <TypesMeta projectId={project.id} />}
    {section === "labels" && <LabelsMeta projectId={project.id} />}
    {section === "repositories" && <ReposMeta projectId={project.id} />}
    {section === "services" && <ServicesMeta projectId={project.id} />}
  </>
);
