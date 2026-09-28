import { useState } from "react";

import { AccountMenuActions } from "@/components/AccountMenuActions";
import { AccountMenuTrigger } from "@/components/AccountMenuTrigger";
import { Popover } from "@/components/ui/popover";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface AccountMenuProps {
  collapsed: boolean;
}

// Opens upward (side="top"): the trigger sits at the sidebar's bottom, so opening down would clip.
export const AccountMenu = ({ collapsed }: AccountMenuProps) => {
  const [open, setOpen] = useState(false);
  // Shares OnboardingGate's cache key, so this never triggers its own fetch.
  const { data: me } = useFetchMe();
  const user = me?.user;
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: role } = useFetchMyRole(selectedWorkspaceId);

  if (!user) {
    return null;
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <AccountMenuTrigger user={user} role={role} collapsed={collapsed} />
      <AccountMenuActions onClose={() => setOpen(false)} />
    </Popover>
  );
};
