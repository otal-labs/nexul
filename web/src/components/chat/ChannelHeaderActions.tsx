import { Settings } from "lucide-react";

import type { Conversation } from "@nexul/client-core/chat";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { PersonAvatar } from "@/components/PersonAvatar";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useChannelRowActions } from "@/hooks/useChannelRowActions";

const SHOWN_MEMBERS = 3;

const MemberStack = ({ channel, onOpen }: { channel: Conversation; onOpen: (() => void) | undefined }) => {
  const resolvePerson = usePersonLookup(channel.workspace_id);
  const ids = channel.participant_ids ?? [];
  const label = `${ids.length} ${ids.length === 1 ? "member" : "members"}`;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={onOpen}
          disabled={!onOpen}
          aria-label={label}
          className="flex h-7 items-center gap-1.5 rounded-md px-1.5 text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent hover:text-foreground disabled:hover:bg-transparent"
        >
          <span className="flex -space-x-1">
            {ids.slice(0, SHOWN_MEMBERS).map((id) => (
              <PersonAvatar key={id} login={resolvePerson(id).login} name={resolvePerson(id).display_name} src={resolvePerson(id).avatar_url} className="size-5 text-[10px] ring-2 ring-panel" />
            ))}
          </span>
          {ids.length > SHOWN_MEMBERS && <span className="translate-y-px font-mono text-xs tabular-nums">+{ids.length - SHOWN_MEMBERS}</span>}
        </button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
};

// A channel's way into what the sidebar row's menu already offers: its members (a private channel) and its settings.
export const ChannelHeaderActions = ({ channel }: { channel: Conversation }) => {
  const { onSettings, settingsDialog } = useChannelRowActions(channel);
  return (
    <div className="-mr-1.5 ml-auto flex shrink-0 items-center gap-1">
      {channel.private && <MemberStack channel={channel} onOpen={onSettings} />}
      {onSettings && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Button type="button" variant="ghost" size="icon" aria-label="Channel settings" className="size-7 text-muted-foreground hover:text-foreground" onClick={onSettings}>
              <Settings className="size-4" aria-hidden />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Channel settings</TooltipContent>
        </Tooltip>
      )}
      {settingsDialog}
    </div>
  );
};
