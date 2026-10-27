import type { LucideIcon } from "lucide-react-native";
import Box from "lucide-react-native/icons/box";
import FileText from "lucide-react-native/icons/file-text";
import GitPullRequest from "lucide-react-native/icons/git-pull-request";
import Hash from "lucide-react-native/icons/hash";
import Ticket from "lucide-react-native/icons/ticket";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import type { SearchHit, SearchKind } from "@nexul/client-core/search";

import { Text } from "@/components/ui/text";

// The kinds the phone has a screen for; automations and memories stay on the web.
export const phoneSearchKinds: SearchKind[] = ["ticket", "doc", "pull_request", "stack", "channel"];

const kindIcon: Partial<Record<SearchKind, LucideIcon>> = { ticket: Ticket, doc: FileText, pull_request: GitPullRequest, stack: Box, channel: Hash };

interface SearchRowProps {
  hit: SearchHit;
  onPress: (hit: SearchHit) => void;
}

export const SearchRow = ({ hit, onPress }: SearchRowProps) => {
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  const Icon = kindIcon[hit.type] ?? FileText;
  const meta = hit.hint ?? hit.status_label;
  return (
    <Pressable role="button" onPress={() => onPress(hit)} className="min-h-14 flex-row items-center gap-3.5 px-5 py-2.5 active:bg-accent">
      <Icon size={18} color={String(mutedForeground)} />
      <View className="min-w-0 flex-1">
        <Text numberOfLines={2} className="text-[15px] leading-5">
          {hit.type === "channel" ? `#${hit.title}` : hit.title}
        </Text>
      </View>
      {meta && <Text className="font-mono text-xs text-muted-foreground">{meta}</Text>}
    </Pressable>
  );
};
