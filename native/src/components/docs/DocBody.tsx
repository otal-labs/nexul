import { useMemo } from "react";
import { View } from "react-native";

import { DocBlockNode } from "@/components/docs/DocBlockNode";
import { Text } from "@/components/ui/text";
import { parseRichBody } from "@/models/Doc";

interface DocBodyProps {
  body: string;
}

// Doc bodies are canonical Tiptap JSON (ADR 0026); a non-JSON legacy body (none in production data today)
// renders as one paragraph rather than round-tripping through a markdown parser. Swap this for the shared
// markdown renderer once Chat lands one; the node walk above stays the reference for the node set to support.
export const DocBody = ({ body }: DocBodyProps) => {
  const blocks = useMemo(() => parseRichBody(body), [body]);
  return (
    <View className="gap-3">
      {!blocks && <Text>{body}</Text>}
      {blocks && blocks.map((node, index) => <DocBlockNode key={index} node={node} />)}
    </View>
  );
};
