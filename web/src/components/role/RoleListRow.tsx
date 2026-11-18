import { CrownIcon } from "lucide-react";

import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

interface RoleListRowProps {
  id: string;
  name: string;
  line: string;
  selected: boolean;
  // How many hold it; left out where the viewer isn't shown the Team.
  count?: number | undefined;
  owner?: boolean;
}

export const RoleListRow = ({ id, name, line, selected, count, owner = false }: RoleListRowProps) => {
  const wsPath = useWorkspacePath();
  return (
    <ListPaneRow
      to={wsPath(`/access/roles/${id}`)}
      title={name}
      snippet={line}
      selected={selected}
      {...(owner && { leading: <CrownIcon aria-label="Owner" className="size-3.5 text-muted-foreground" /> })}
      {...(count !== undefined && { meta: <span className="font-mono text-micro text-muted-foreground tabular-nums">{count}</span> })}
    />
  );
};
