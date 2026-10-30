import { Link } from "react-router";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { cn } from "@/lib/utils";

export interface Crumb {
  label: string;
  to?: string;
}

interface PageBreadcrumbProps {
  crumbs: Crumb[];
  className?: string;
}

interface CrumbItemProps {
  crumb: Crumb;
  first: boolean;
  last: boolean;
}

// A middle crumb is cut to 40 characters in text, not by the browser, which drops a space before its ellipsis and leaves a gap.
const CRUMB_CHARS = 40;
// A crumb's label can still be loading (a workspace crumb before its name and slug arrive).
const crumbText = (label: string | undefined, last: boolean) =>
  !label || last || label.length <= CRUMB_CHARS ? label : `${label.slice(0, CRUMB_CHARS - 1).trimEnd()}…`;

const CrumbItem = ({ crumb, first, last }: CrumbItemProps) => (
  <>
    {!first && <BreadcrumbSeparator className="[&>svg]:size-3" />}
    <BreadcrumbItem className={cn("min-w-0", !last && "max-w-[40ch] min-w-8 shrink-[100]")} title={crumb.label}>
      {crumb.to && (
        <BreadcrumbLink asChild className="truncate transition-colors duration-150 ease-standard">
          <Link to={crumb.to}>{crumbText(crumb.label, last)}</Link>
        </BreadcrumbLink>
      )}
      {!crumb.to && <BreadcrumbPage className="truncate text-muted-foreground">{crumbText(crumb.label, last)}</BreadcrumbPage>}
    </BreadcrumbItem>
  </>
);

export const PageBreadcrumb = ({ crumbs, className }: PageBreadcrumbProps) => (
  <Breadcrumb className={className}>
    <BreadcrumbList className="flex-nowrap gap-1.5 text-xs sm:gap-1.5">
      {crumbs.map((crumb, index) => (
        <CrumbItem
          key={`${index}-${crumb.label}`}
          crumb={crumb}
          first={index === 0}
          last={index === crumbs.length - 1}
        />
      ))}
    </BreadcrumbList>
  </Breadcrumb>
);
