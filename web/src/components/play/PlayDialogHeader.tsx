import { TabUnderline } from "@/components/ActiveIndicator";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { usePlayDialogStore, type PlayDialogTab } from "@/stores/playDialogStore";

interface PlayDialogHeaderProps {
  title: string;
}

// Line tabs held in the dialog rather than the path, since a dialog has none; pinned under the title while the body scrolls.
export const PlayDialogHeader = ({ title }: PlayDialogHeaderProps) => {
  const tab = usePlayDialogStore((s) => s.tab);
  const setTab = usePlayDialogStore((s) => s.setTab);
  const leaveGuard = usePlayDialogStore((s) => s.leaveGuard);

  const select = async (next: PlayDialogTab) => {
    if (next !== tab && leaveGuard && !(await leaveGuard())) return;
    setTab(next);
  };

  return (
    <div className="space-y-3">
      <p className="text-base leading-snug font-semibold [overflow-wrap:anywhere]">{title}</p>
      <Tabs value={tab} onValueChange={(value) => void select(value as PlayDialogTab)}>
        <TabsList
          variant="line"
          aria-label="Play settings"
          className="relative isolate -mr-6 w-[calc(100%+1.5rem)] justify-start border-b border-border p-0"
        >
          <TabUnderline />
          <TabsTrigger value="play" className="flex-none px-3 after:hidden">
            Play
          </TabsTrigger>
          <TabsTrigger value="auto" className="flex-none px-3 after:hidden">
            Auto plays
          </TabsTrigger>
        </TabsList>
      </Tabs>
    </div>
  );
};
