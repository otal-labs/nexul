import { createContext, useContext } from "react";

import type { Crumb } from "@/components/PageBreadcrumb";

// Set by a page that shows another page's record inside it (the Inbox): the record's crumbs lead back to that page instead.
export const EmbeddedCrumbsContext = createContext<Crumb[] | undefined>(undefined);

export const useEmbeddedCrumbs = (own: Crumb[]): Crumb[] => useContext(EmbeddedCrumbsContext) ?? own;
