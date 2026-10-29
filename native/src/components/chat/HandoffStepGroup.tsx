import CircleX from "lucide-react-native/icons/circle-x";
import FilePen from "lucide-react-native/icons/file-pen";
import FileSearch from "lucide-react-native/icons/file-search";
import Plug from "lucide-react-native/icons/plug";
import SquareTerminal from "lucide-react-native/icons/square-terminal";
import Wrench from "lucide-react-native/icons/wrench";
import type { LucideIcon } from "lucide-react-native";
import { View } from "react-native";

import { groupHeadline, isFailedStep, isMcpTool, stepAct, stepPhrase, type StepAct, type StepGroup } from "@nexul/client-core/agentSteps";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import type { HandoffStep } from "@/models/Chat";

interface HandoffStepGroupProps {
  group: StepGroup<HandoffStep>;
  // root is the run's directory, cut from the front of its commands.
  root: string | null;
  // running reads an unanswered call in the present.
  running: boolean;
}

// The web's step icons: a file search for reads, a file pen for edits, a terminal for a command, a plug for an MCP
// call, a wrench for any other tool.
const iconOf = (act: StepAct, tool: string | undefined): LucideIcon => {
  if (act === "explore") return FileSearch;
  if (act === "edit") return FilePen;
  if (act === "command") return SquareTerminal;
  return isMcpTool(tool) ? Plug : Wrench;
};

const StepIcon = ({ act, tool }: { act: StepAct; tool: string | undefined }) => <Icon as={iconOf(act, tool)} className="size-3.5 text-muted-foreground" />;

const Failed = () => (
  <View className="flex-row items-center gap-1">
    <Icon as={CircleX} className="size-3.5 text-destructive" />
    <Text className="text-xs text-muted-foreground">failed</Text>
  </View>
);

// A search reads as its quoted pattern, a file as its name alone, a command as commandText reads it.
const nameOf = (step: HandoffStep, root: string | null): string => {
  const { verb, object } = stepPhrase(step, false, root);
  if (stepAct(step) === "explore" && verb.startsWith("Searched")) return `“${object}”`;
  return object;
};

const NameLine = ({ step, root }: { step: HandoffStep; root: string | null }) => {
  const failed = isFailedStep(step);
  const name = nameOf(step, root);
  return (
    <View accessible aria-label={failed ? `${name}, failed` : name} className="flex-row items-center gap-1.5 pl-[22px]">
      <Text numberOfLines={1} className={cn("shrink font-mono text-xs leading-5 text-foreground/80", failed && "text-destructive")}>
        {name}
      </Text>
      {failed && <Failed />}
    </View>
  );
};

type ListedAct = "explore" | "edit" | "command";

// Files always list under a count; a command does once a neighbour joins it, and stands as one row alone.
const listedAct = (group: StepGroup<HandoffStep>): ListedAct | null => {
  if (group.act === "explore" || group.act === "edit") return group.act;
  if (group.act === "command" && group.entries.length > 1) return "command";
  return null;
};

const StepList = ({ act, entries, root }: { act: ListedAct; entries: HandoffStep[]; root: string | null }) => (
  <View className="gap-0.5 py-1">
    <View className="flex-row items-center gap-2">
      <StepIcon act={act} tool={undefined} />
      <Text className="text-sm text-muted-foreground">{groupHeadline(act, entries, false)}</Text>
    </View>
    {entries.map((step, i) => (
      <NameLine key={`${step.call_id ?? step.at}-${i}`} step={step} root={root} />
    ))}
  </View>
);

const StepLine = ({ step, root, running }: { step: HandoffStep; root: string | null; running: boolean }) => {
  const { verb, object } = stepPhrase(step, running && step.kind === "tool_call", root);
  const failed = isFailedStep(step);
  return (
    <View accessible aria-label={`${verb} ${object}${failed ? ", failed" : ""}`} className="flex-row items-center gap-2 py-1">
      <StepIcon act={stepAct(step)} tool={step.tool} />
      <Text className="text-sm text-muted-foreground">{verb}</Text>
      <Text numberOfLines={1} className="shrink font-mono text-xs text-foreground/80">
        {object}
      </Text>
      {failed && <Failed />}
    </View>
  );
};

// One step group of a helper's turn, the web's rows in the phone's type: neighbouring files or commands under a count,
// one name a line; a lone command or tool as its icon, verb and mono name; the helper's own sentence as text.
export const HandoffStepGroup = ({ group, root, running }: HandoffStepGroupProps) => {
  const first = group.entries[0];
  const listed = listedAct(group);
  return (
    <>
      {listed !== null && <StepList act={listed} entries={group.entries} root={root} />}
      {listed === null && first && group.act === "say" && (
        <Text className={cn("py-1 text-sm text-foreground/85", first.kind === "note" && "italic text-muted-foreground")}>{first.detail || first.summary}</Text>
      )}
      {listed === null && first && group.act !== "say" && <StepLine step={first} root={root} running={running} />}
    </>
  );
};
