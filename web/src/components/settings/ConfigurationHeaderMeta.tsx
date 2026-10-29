import type { SettingsSection } from "@/components/settings/SettingsNav";
import { useFetchInterviewTemplate } from "@/hooks/MemoryHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const count = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;

const GeneralMeta = () => {
  const workspace = useSelectedWorkspace();
  return <span className="font-mono">{workspace && `/${workspace.slug}`}</span>;
};

const InterviewMeta = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: template } = useFetchInterviewTemplate(workspaceId);
  return <span className="tabular-nums">{template && count(template.questions.length, "question")}</span>;
};

const MentionsMeta = () => {
  const workspace = useSelectedWorkspace();
  const fields = workspace?.mention_chip_template.match(/\{ticket\.\w+\}/g)?.length ?? 0;
  return <span className="tabular-nums">{workspace && count(fields, "field")}</span>;
};

const TeamMeta = () => {
  const { data: team } = useFetchTeam();
  return <span className="tabular-nums">{team && count(team.people.length, "person", "people")}</span>;
};

interface ConfigurationHeaderMetaProps {
  section: SettingsSection;
}

// Identity and counts for the open section, never its state; every section has a line, so the header keeps its height.
export const ConfigurationHeaderMeta = ({ section }: ConfigurationHeaderMetaProps) => (
  <span className="inline-flex min-h-5 items-center">
    {(section === "general" || section === "danger") && <GeneralMeta />}
    {section === "interview" && <InterviewMeta />}
    {section === "mentions" && <MentionsMeta />}
    {section === "team" && <TeamMeta />}
  </span>
);
