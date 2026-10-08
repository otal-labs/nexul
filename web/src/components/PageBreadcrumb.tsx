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

const CrumbItem = ({ crumb, first, last }: CrumbItemProps) => (
  <>
    {!first && <BreadcrumbSeparator className="[&>svg]:size-3" />}
    <BreadcrumbItem className={cn("min-w-0", !last && "max-w-48 min-w-8 shrink-[100]")}>
      {crumb.to && (
        <BreadcrumbLink asChild className="truncate duration-150 ease-standard">
          <Link to={crumb.to}>{crumb.label}</Link>
        </BreadcrumbLink>
      )}
      {!crumb.to && <BreadcrumbPage className="truncate text-muted-foreground">{crumb.label}</BreadcrumbPage>}
    </BreadcrumbItem>
  </>
);

export const PageBreadcrumb = ({ crumbs, className }: PageBreadcrumbProps) => (
  <Breadcrumb className={className}>
    <BreadcrumbList className="flex-nowrap gap-1.5 font-mono text-xs sm:gap-1.5">
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
