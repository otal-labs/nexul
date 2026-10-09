import { Text } from "@/components/ui/text";

// An empty list inside a section: one muted sentence where the rows would be.
export const EmptyRow = ({ message }: { message: string }) => (
  <Text className="rounded-xl border border-dashed border-input px-4 py-3.5 text-[13px] text-muted-foreground">{message}</Text>
);
