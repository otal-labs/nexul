import { DocOverviewRow } from "@/components/doc/DocOverviewRow";
import type { AttentionDoc } from "@/components/doc/docsOverviewLists";

interface DocAttentionRowProps {
  item: AttentionDoc;
  projectToken: string;
  folderName: string;
}

export const DocAttentionRow = ({ item: { doc, state }, projectToken, folderName }: DocAttentionRowProps) => (
  <DocOverviewRow
    doc={doc}
    runState={state}
    projectToken={projectToken}
    detail={`${folderName} · ${state === "waiting" ? "Waiting on an answer" : "Agent writing"}`}
  />
);
