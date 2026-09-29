import { Loader2Icon } from "lucide-react";

import { cn } from "@/lib/utils";

const Spinner = ({ className, ...props }: React.ComponentProps<"svg">) => (
  <Loader2Icon
    data-slot="spinner"
    role="status"
    aria-label="Loading"
    className={cn("size-4 animate-spin text-current motion-reduce:animate-none", className)}
    {...props}
  />
);

export { Spinner };
