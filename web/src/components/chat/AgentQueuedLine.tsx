import { Clock3, CornerDownRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useSteerAgentTurn } from "@/hooks/ChatHooks";
import { useAgentStreamStore } from "@/stores/agentStreamStore";

interface AgentQueuedLineProps {
  conversationId: string;
  messageId: string;
}

// Under a message whose Agent turn waits behind the one under way: it has not reached the Agent yet, and Steer sends it now.
export const AgentQueuedLine = ({ conversationId, messageId }: AgentQueuedLineProps) => {
  const queued = useAgentStreamStore((s) => messageId in s.queued);
  const steer = useSteerAgentTurn(conversationId);

  return (
    queued && (
      <div data-slot="queued" className="settle-in flex items-center gap-2 text-xs text-muted-foreground">
        <Clock3 className="size-3.5" aria-hidden />
        <span>Queued, the Agent is busy</span>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="outline" size="sm" className="h-7 px-2 text-xs" loading={steer.isPending} onClick={() => steer.mutate(messageId)}>
              {!steer.isPending && <CornerDownRight aria-hidden />}
              Steer
            </Button>
          </TooltipTrigger>
          <TooltipContent>Send it into the Agent's current turn</TooltipContent>
        </Tooltip>
      </div>
    )
  );
};
