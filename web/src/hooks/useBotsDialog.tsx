import { useState } from "react";

import { BotsDialog } from "@/components/botwebhook/BotsDialog";
import { useAreaAccess } from "@/hooks/AccessHooks";
import type { Conversation } from "@/models/Chat";

// A conversation menu's Bots item, undefined without botwebhook:read; the menu's owner renders botsDialog.
export const useBotsDialog = (conversation: Conversation | undefined, label: string) => {
  const can = useAreaAccess();
  const [open, setOpen] = useState(false);
  return {
    onBots: conversation && can?.("bots") ? () => setOpen(true) : undefined,
    botsDialog: conversation && <BotsDialog conversation={conversation} label={label} open={open} onClose={() => setOpen(false)} />,
  };
};
