import type { SettingsSection } from "@/components/settings/SettingsNav";
import { useFetchInterviewTemplate } from "@/hooks/MemoryHooks";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const count = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;

const join = (parts: (string | false | undefined)[]) => parts.filter(Boolean).join(" · ");

const GeneralMeta = () => {
  const workspace = useSelectedWorkspace();
  return <span className="font-mono">{workspace && `/${workspace.slug}`}</span>;
};

const RolesMeta = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: roles } = useFetchWorkspaceRoles(workspaceId);
  const { data: team } = useFetchTeam();
  const members = team?.people.filter((person) => person.workspaces.some((m) => m.workspace_id === workspaceId)).length;
  return <>{join([roles && count(roles.length, "role"), members !== undefined && count(members, "person", "people")])}</>;
};

const PlaysMeta = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: plays } = useFetchWorkspacePlays(workspaceId);
  const off = plays?.filter((play) => !play.enabled).length ?? 0;
  return <>{join([plays && count(plays.length, "play"), off > 0 && `${off} off`])}</>;
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
    {section === "roles" && <RolesMeta />}
    {section === "plays" && <PlaysMeta />}
    {section === "interview" && <InterviewMeta />}
    {section === "mentions" && <MentionsMeta />}
  </>
);
