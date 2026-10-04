import { Pressable, View } from "react-native";

import { Text } from "@/components/ui/text";
import { DOC_TITLE } from "@/components/docs/prototype/ClarifyProtoData";
import { cn } from "@/lib/utils";

interface MockRow {
  title: string;
  summary: string;
  time: string;
  unread: boolean;
}

const ROWS: MockRow[] = [
  { title: `New questions on ${DOC_TITLE}`, summary: "Round 1 · 5 questions", time: "2m", unread: true },
  { title: "Opening hours for the summer", summary: "doc updated", time: "1h", unread: false },
  { title: "Price list 2027", summary: "mention", time: "yesterday", unread: false },
];

interface ClarifyProtoInboxRowProps {
  row: MockRow;
  onPress: () => void;
}

// NotificationRow's shape with the two-line title a long doc name needs.
const ClarifyProtoInboxRow = ({ row, onPress }: ClarifyProtoInboxRowProps) => (
  <Pressable
    role="button"
    onPress={onPress}
    android_ripple={{ borderless: false }}
    className="min-h-12 flex-row items-start gap-2.5 border-b border-border px-4 py-3"
  >
    <View className={cn("mt-1.5 size-2 shrink-0 rounded-full", row.unread && "bg-primary")} />
    <View className="min-w-0 flex-1">
      <Text numberOfLines={2} className={row.unread ? "font-semibold" : "font-medium text-muted-foreground"}>
        {row.title}
      </Text>
      <Text variant="muted" numberOfLines={1} className="text-xs leading-snug">
        {row.summary}
      </Text>
    </View>
    <Text variant="muted" className="font-mono text-xs leading-snug">
      {row.time}
    </Text>
  </Pressable>
);

// The Inbox as the client finds it after the push; the first row opens the doc on its questions.
export const ClarifyProtoInbox = ({ onOpen }: { onOpen: () => void }) => (
  <View className="-mx-4 border-t border-border">
    {ROWS.map((row) => (
      <ClarifyProtoInboxRow key={row.title} row={row} onPress={row.unread ? onOpen : () => {}} />
    ))}
  </View>
);
