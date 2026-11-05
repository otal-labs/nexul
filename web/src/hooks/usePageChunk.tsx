import type { ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";

import { loadPages, pagesLoaded } from "@/pageChunks";

// Whether a page renders without suspending, loading its chunk once enabled; a Suspense fallback's reveal waits 300ms.
export const usePageChunk = (key: string, page: ReactNode, enabled: boolean): boolean => {
  const loaded = pagesLoaded(page);
  // eslint-disable-next-line @tanstack/query/exhaustive-deps -- the route id names the page; an element can't be a key
  const { isSuccess } = useQuery({
    queryKey: ["pageChunk", key],
    queryFn: () => loadPages(page),
    enabled: enabled && !loaded,
    staleTime: Infinity,
  });
  return loaded || isSuccess;
};
