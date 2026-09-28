import { Text } from "@/components/ui/text";

interface MarkdownTextProps {
  content: string;
}

// Isolated on purpose: renders the raw markdown as plain wrapped text for now. Swap the body for the shared
// renderer once Chat lands one, without touching any other ticket-screen code.
export const MarkdownText = ({ content }: MarkdownTextProps) => <Text className="leading-6">{content}</Text>;
