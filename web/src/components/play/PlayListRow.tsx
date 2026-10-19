import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { usePlayMenu } from "@/hooks/usePlayMenu";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { PLAY_STAGE_LABELS, type Play } from "@/models/Play";

interface PlayListRowProps {
  play: Play;
  selected: boolean;
}

const factsOf = (play: Play) =>
  [play.type === "ticket" && play.show_when_stage && PLAY_STAGE_LABELS[play.show_when_stage], !play.enabled && "Disabled"]
    .filter(Boolean)
    .join(" · ");

export const PlayListRow = ({ play, selected }: PlayListRowProps) => {
  const wsPath = useWorkspacePath();
  const menu = usePlayMenu(play);
  return <ListPaneRow to={wsPath(`/plays/${play.id}`)} title={play.label} snippet={factsOf(play)} selected={selected} menu={menu} />;
};
