import { lazy, Suspense, useState } from "react";
import { useSearchParams } from "react-router";

import { COMPUTERS_PARAM } from "@/models/Agent";

const ComputersDialog = lazy(() => import("@/components/computers/ComputersDialog").then((m) => ({ default: m.ComputersDialog })));

// Mounted once in the app shell: the dialog's code loads the first time a link sets the param, then stays so closing can animate out.
export const LazyComputersDialog = () => {
  const [params] = useSearchParams();
  const [wanted, setWanted] = useState(false);
  if (params.has(COMPUTERS_PARAM) && !wanted) setWanted(true);
  if (!wanted) return null;

  return (
    <Suspense fallback={null}>
      <ComputersDialog />
    </Suspense>
  );
};
