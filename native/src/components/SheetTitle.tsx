import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface SheetTitleProps {
  title: string;
  className?: string;
}

export const SheetTitle = ({ title, className }: SheetTitleProps) => (
  <Text role="heading" className={cn("px-4 pb-3 pt-5 text-lg font-semibold", className)} numberOfLines={1}>
    {title}
  </Text>
);
