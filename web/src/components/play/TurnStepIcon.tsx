import { FilePen, FileSearch, Plug, SquareTerminal, Wrench } from "lucide-react";

import { isMcpTool, type StepAct } from "@nexul/client-core/agentSteps";

interface TurnStepIconProps {
  act: StepAct;
  tool: string | undefined;
}

// A step row's leading icon, muted, in the column the turn's header text starts at.
export const TurnStepIcon = ({ act, tool }: TurnStepIconProps) => {
  const className = "size-3.5 shrink-0 text-muted-foreground";
  return (
    <>
      {act === "explore" && <FileSearch className={className} aria-hidden />}
      {act === "edit" && <FilePen className={className} aria-hidden />}
      {act === "command" && <SquareTerminal className={className} aria-hidden />}
      {act === "tool" && isMcpTool(tool) && <Plug className={className} aria-hidden />}
      {act === "tool" && !isMcpTool(tool) && <Wrench className={className} aria-hidden />}
    </>
  );
};
