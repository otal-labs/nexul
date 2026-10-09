import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface SheetTitleProps {
  title: string;
  className?: string;
}

// What a sheet acts on, in the dialog title's 17pt semibold.
export const SheetTitle = ({ title, className }: SheetTitleProps) => (
  <Text role="heading" className={cn("px-5 pb-3 pt-6 text-[17px] font-semibold", className)} numberOfLines={2}>
    {title}
  </Text>
);
