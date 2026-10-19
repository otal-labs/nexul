import { PageHeader } from "@/components/PageHeader";
import { ActionButton } from "@/components/action/ActionButton";
import { RowActions } from "@/components/listpane/RowActions";
import { offersMenu } from "@/components/listpane/rowMenu";
import { useEmbeddedCrumbs } from "@/hooks/useEmbeddedCrumbs";
import { useEntityActions } from "@/hooks/useEntityActions";
import { usePlayDialogs } from "@/hooks/usePlayDialogs";
import { usePlayMenu } from "@/hooks/usePlayMenu";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { PLAY_TYPE_LABELS, type Play } from "@/models/Play";

interface PlayDetailHeaderProps {
  play: Play;
}

export const PlayDetailHeader = ({ play }: PlayDetailHeaderProps) => {
  const wsPath = useWorkspacePath();
  const crumbs = useEmbeddedCrumbs([{ label: "Plays", to: wsPath("/plays") }, { label: `${PLAY_TYPE_LABELS[play.type]} play` }]);
  const answer = useEntityActions("play", play.id);
  const { edit } = usePlayDialogs();
  const menu = { ...usePlayMenu(play), onEdit: undefined };

  return (
    <PageHeader
      crumbs={crumbs}
      title={play.label}
      meta={play.description && <span className="max-w-prose">{play.description}</span>}
      actions={
        <>
          <ActionButton permission="plays:write" answer={answer("plays:write")} verb="edit plays" variant="outline" size="sm" onClick={() => void edit(play)}>
            Edit
          </ActionButton>
          {offersMenu(menu, !!menu.permissions) && <RowActions itemLabel={play.label} {...menu} />}
        </>
      }
    />
  );
};
