import type { SettingsSection } from "@/components/settings/SettingsNav";
import { useFetchInterviewTemplate } from "@/hooks/MemoryHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const count = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;

const join = (parts: (string | false | undefined)[]) => parts.filter(Boolean).join(" · ");

const GeneralMeta = () => {
  const workspace = useSelectedWorkspace();
  return <span className="font-mono">{workspace && `/${workspace.slug}`}</span>;
};

const InterviewMeta = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: template } = useFetchInterviewTemplate(workspaceId);
  return <>{template && join([count(template.questions.length, "question"), template.edited ? "edited here" : "following the instance"])}</>;
};

const MentionsMeta = () => {
  const workspace = useSelectedWorkspace();
  return <>{workspace && (workspace.mention_chip_template_edited ? "Edited here" : "Following the instance")}</>;
};

interface ConfigurationHeaderMetaProps {
  section: SettingsSection;
}

// Facts about the open section, not a tagline; each reads the query its section already loads.
export const ConfigurationHeaderMeta = ({ section }: ConfigurationHeaderMetaProps) => (
  <>
    {section === "general" && <GeneralMeta />}
    {section === "interview" && <InterviewMeta />}
    {section === "mentions" && <MentionsMeta />}
  </>
);
