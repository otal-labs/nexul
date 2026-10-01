import { useState } from "react";

import { Tabs, TabsContent, TabsList } from "@/components/ui/tabs";
import { EmptyRow } from "@/components/EmptyRow";
import { TeamAddWorkspacePopover } from "@/components/team/TeamAddWorkspacePopover";
import { TeamWorkspacePanel } from "@/components/team/TeamWorkspacePanel";
import { TeamWorkspaceTab } from "@/components/team/TeamWorkspaceTab";
import { useMemberDraft } from "@/hooks/useMemberDraft";
import { draftedMembership, type MemberStep } from "@/models/MemberDraft";
import type { TeamWorkspace } from "@/models/Team";

interface TeamWorkspaceTabsProps {
  workspaces: TeamWorkspace[];
  steps: MemberStep[];
}

// One tab per workspace the person is in, then the ones added here, with + after the last like a browser's tab strip.
export const TeamWorkspaceTabs = ({ workspaces, steps }: TeamWorkspaceTabsProps) => {
  const { person, draft } = useMemberDraft();
  const [selected, setSelected] = useState<string>();
  const stored = person.workspaces.map((membership) => membership.workspace_id);
  const added = Object.keys(draft).filter((id) => draft[id]?.added && !stored.includes(id));
  const tabs = [...stored, ...added].flatMap((id) => {
    const workspace = workspaces.find((candidate) => candidate.id === id);
    const membership = workspace && draftedMembership(person.workspaces.find((m) => m.workspace_id === id), workspace, draft[id]);
    return workspace && membership ? [{ workspace, membership }] : [];
  });
  const active = tabs.find((tab) => tab.workspace.id === selected) ?? tabs[0];
  const pending = new Set(steps.map((step) => step.workspaceId));
  const removed = person.status === "removed";

  return (
    <Tabs value={active?.workspace.id ?? ""} onValueChange={setSelected} className="min-h-0 flex-1 gap-0">
      <div className="flex items-end gap-1 border-b border-border px-6">
        <TabsList variant="line" aria-label="Workspaces" className="h-auto min-w-0 justify-start gap-0.5 overflow-x-auto overflow-y-hidden p-0">
          {tabs.map(({ workspace }) => (
            <TeamWorkspaceTab key={workspace.id} value={workspace.id} name={workspace.name} pending={pending.has(workspace.id)} />
          ))}
        </TabsList>
        <TeamAddWorkspacePopover workspaces={workspaces} onAdded={setSelected} />
      </div>
      {tabs.length === 0 && (
        <div className="px-6 py-4">
          <EmptyRow>{removed ? "Restore the account to give it workspace access." : "Not a member of any workspace you can see."}</EmptyRow>
        </div>
      )}
      {tabs.map(({ workspace, membership }) => (
        <TabsContent key={workspace.id} value={workspace.id} className="min-h-0 overflow-y-auto px-6 py-4">
          <TeamWorkspacePanel workspace={workspace} membership={membership} />
        </TabsContent>
      ))}
    </Tabs>
  );
};
