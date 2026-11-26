import { HeaderFacts } from "@/components/HeaderFacts";
import type { SettingsSection } from "@/components/settings/SettingsNav";
import { useFetchImports } from "@/hooks/ImportHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchInterviewTemplate } from "@/hooks/MemoryHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const count = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;

// What the General card doesn't show: how much the workspace holds and since when; its URL is the card's first field.
const GeneralMeta = () => {
  const workspace = useSelectedWorkspace();
  const { data: projects } = useFetchProjects();
  const since = workspace?.created_at && new Date(workspace.created_at).toLocaleDateString(undefined, { dateStyle: "medium" });
  return <span className="tabular-nums"><HeaderFacts parts={[projects && count(projects.length, "project"), since && `Since ${since}`]} /></span>;
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

const ImportMeta = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: imports } = useFetchImports(workspaceId);
  return <span className="tabular-nums">{imports && count(imports.length, "import")}</span>;
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
    {section === "import" && <ImportMeta />}
  </span>
);
