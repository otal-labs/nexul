import { useRouter } from "expo-router";
import { ChevronsUpDown } from "lucide-react-native";
import { Pressable } from "react-native";
import { useCSSVariable } from "uniwind";

import { ProjectMark } from "@/components/ProjectMark";
import { ScreenHeader } from "@/components/ScreenHeader";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import type { Project } from "@/models/Project";

interface BoardHeaderProps {
  project: Project | undefined;
  ticketCount: number | undefined;
}

const countLine = (count: number | undefined) => (count === undefined ? undefined : `${count} ${count === 1 ? "ticket" : "tickets"}`);

// The project mark beside its name in the display face; the switch opens the project picker sheet.
export const BoardHeader = ({ project, ticketCount }: BoardHeaderProps) => {
  const router = useRouter();
  const workspace = useSelectedWorkspace();
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <ScreenHeader
      eyebrow={workspace?.name}
      title={project?.name ?? "Board"}
      meta={countLine(ticketCount)}
      leading={project && <ProjectMark prefix={project.prefix} />}
      action={
        project && (
          <Pressable
            role="button"
            aria-label="Switch project"
            onPress={() => router.push("/board/project-picker")}
            className="size-11 items-center justify-center rounded-md border border-input active:bg-accent"
          >
            <ChevronsUpDown size={18} color={String(muted)} />
          </Pressable>
        )
      }
    />
  );
};
