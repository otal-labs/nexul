import { useState } from "react";
import { SparklesIcon } from "lucide-react";

import { ListPaneEmpty, ListPaneNoMatch } from "@/components/listpane/ListPaneEmpty";
import { ListPaneGroup } from "@/components/listpane/ListPaneGroup";
import { ListPaneHeader } from "@/components/listpane/ListPaneHeader";
import { groupPlaysByType } from "@/components/play/playGroups";
import { PlayListRow } from "@/components/play/PlayListRow";
import { usePlayDialogs } from "@/hooks/usePlayDialogs";
import { PLAY_TYPE_LABELS, type WorkspacePlay } from "@/models/Play";

interface PlaysListPaneProps {
  plays: WorkspacePlay[];
  selectedId: string | undefined;
}

const matches = (play: WorkspacePlay, query: string) => [play.label, play.description].some((text) => text.toLowerCase().includes(query));

export const PlaysListPane = ({ plays, selectedId }: PlaysListPaneProps) => {
  const [search, setSearch] = useState("");
  const { create } = usePlayDialogs();
  const query = search.trim().toLowerCase();
  const groups = groupPlaysByType(query === "" ? plays : plays.filter((play) => matches(play, query)));

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ListPaneHeader
        title="Plays"
        count={plays.length}
        newLabel="New play"
        newPermission="plays:write"
        newVerb="create plays"
        onNew={() => void create()}
        search={search}
        onSearch={setSearch}
      />
      <div className="scroll-edge min-h-0 flex-1 overflow-y-auto pb-6">
        {plays.length === 0 && <ListPaneEmpty icon={SparklesIcon} message="No plays yet" />}
        {plays.length > 0 && groups.length === 0 && <ListPaneNoMatch onClear={() => setSearch("")} />}
        {groups.map((group) => (
          <ListPaneGroup key={group.type} label={PLAY_TYPE_LABELS[group.type]} count={group.plays.length}>
            {group.plays.map((play) => (
              <PlayListRow key={play.id} play={play} selected={play.id === selectedId} />
            ))}
          </ListPaneGroup>
        ))}
      </div>
    </div>
  );
};
