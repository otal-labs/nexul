import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { ComputersSection } from "@/components/settings/ComputersSection";
import { PairingDefaultsSection } from "@/components/settings/PairingDefaultsSection";
import { PairingProjectsSection } from "@/components/settings/PairingProjectsSection";
import { useSkillsOutdated } from "@/hooks/ComputerSetupHooks";
import { SKILLS_OUTDATED } from "@/models/Pairing";

// Computers comes first so /settings/pairing?setup=<id> (no tab segment) opens the setup summary it names.
export const PairingPanel = () => {
  const skillsOutdated = useSkillsOutdated();
  return (
    <PageTabs
      label="Computers settings"
      tabs={[
        { value: "computers", label: "Computers", dot: skillsOutdated ? SKILLS_OUTDATED : undefined },
        { value: "projects", label: "Projects" },
        { value: "defaults", label: "Defaults" },
      ]}
    >
      <PageTabsContent value="computers">
        <ComputersSection />
      </PageTabsContent>
      <PageTabsContent value="projects">
        <PairingProjectsSection />
      </PageTabsContent>
      <PageTabsContent value="defaults">
        <PairingDefaultsSection />
      </PageTabsContent>
    </PageTabs>
  );
};
