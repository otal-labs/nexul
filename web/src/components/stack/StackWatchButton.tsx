import { EyeIcon, EyeOffIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useFetchStack, useSetStackWatching } from "@/hooks/StackHooks";
import { useStackActions } from "@/hooks/useStackActions";

export const StackWatchButton = ({ stackId }: { stackId: string }) => {
  const { data: stack } = useFetchStack(stackId);
  const answer = useStackActions(stack);
  const setWatching = useSetStackWatching(stackId);
  const watching = stack?.watching === true;
  // A stack outside every project has no one to tell.
  if (!stack?.project_id) return null;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <ActionButton
          permission="stacks:read"
          projectId={stack.project_id}
          answer={answer("stacks:read")}
          verb="watch stacks"
          variant="outline"
          size="sm"
          icon={watching ? EyeIcon : EyeOffIcon}
          aria-pressed={watching}
          onClick={() => setWatching.mutate(!watching)}
        >
          {watching ? "Watching" : "Watch"}
        </ActionButton>
      </TooltipTrigger>
      <TooltipContent>Get a notification when a service in this stack crashes and when it comes back.</TooltipContent>
    </Tooltip>
  );
};
