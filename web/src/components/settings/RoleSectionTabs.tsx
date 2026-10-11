import { Tabs, TabsContent, TabsList } from "@/components/ui/tabs";
import { BrowserTab } from "@/components/BrowserTab";
import { RoleSectionList } from "@/components/settings/RoleSectionList";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useRoleDraft } from "@/hooks/useRoleDraft";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import type { PermissionArea } from "@/models/Permission";
import { ROLE_SECTIONS, changedIn, sectionDomains, sectionLine, sectionTally } from "@/models/RoleSection";

interface RoleSectionTabsProps {
  tab: PermissionArea;
  onTab: (tab: PermissionArea) => void;
}

// One tab per area, each with its allowed count and a dot while it holds changes; only the open tab's list scrolls.
export const RoleSectionTabs = ({ tab, onTab }: RoleSectionTabsProps) => {
  const { data: catalog = [] } = useFetchPermissionCatalog();
  const workspace = useSelectedWorkspace();
  const { actions, loaded } = useRoleDraft();
  const sections = ROLE_SECTIONS.map((section) => {
    const domains = sectionDomains(catalog, section.area);
    return { section, domains, tally: sectionTally(domains, actions), changed: changedIn(catalog, section.area, actions, loaded) };
  });

  return (
    <Tabs value={tab} onValueChange={(next) => onTab(next as PermissionArea)} className="min-h-0 flex-1 gap-0">
      <div className="border-b border-border px-6">
        <TabsList variant="line" aria-label="Where the permissions apply" className="h-auto justify-start gap-0.5 p-0">
          {sections.map(({ section, tally, changed }) => (
            <BrowserTab key={section.area} value={section.area} pending={changed}>
              {section.title}
              <span aria-hidden className="font-mono text-[11px] tabular-nums text-muted-foreground">
                {tally.allowed}/{tally.total}
              </span>
              <span className="sr-only">
                , {tally.allowed} of {tally.total} allowed
              </span>
            </BrowserTab>
          ))}
        </TabsList>
      </div>
      {sections.map(({ section, domains }) => (
        <TabsContent key={section.area} value={section.area} className="min-h-0 overflow-y-auto px-6 pt-4 pb-6">
          <RoleSectionList section={section} line={sectionLine(section.area, workspace?.name ?? "")} domains={domains} />
        </TabsContent>
      ))}
    </Tabs>
  );
};
