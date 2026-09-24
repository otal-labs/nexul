import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { McpSnippetCard } from "@/components/settings/McpSnippetCard";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchMemorySkill } from "@/hooks/MemorySkillHooks";

export const MemorySkillSection = () => {
  const { data: skill, error, isPending } = useFetchMemorySkill();

  return (
    <SettingsCard
      id="pairing-memory-skill"
      title="Agent memory skill"
      description="Copy this into your own T3 Code skills to teach any session Nexul's shared memory protocol — the same one Agent follows in chat."
    >
      {isPending && <LoadingDisplay label="Loading the skill" className="justify-start p-0" />}
      {error && <ErrorDisplay error={error} title="Skill unavailable" className="p-3" />}
      {skill && <McpSnippetCard label="nexul-memory" filename="SKILL.md" snippet={skill} />}
    </SettingsCard>
  );
};
