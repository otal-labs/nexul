import { useEffect, useRef } from "react";

import { SetupFinishStatus } from "@/components/setup/SetupFinishStatus";
import { useSetInstanceUrl } from "@/hooks/SetupHooks";

// Desktop installs: http://localhost:<port> is already a valid GitHub callback, so it is stored without a domain step.
export const SetupLocalFinish = () => {
  const finish = useSetInstanceUrl();
  const { mutate } = finish;
  const started = useRef(false);

  // Must run exactly once; the ref guards against StrictMode double-invoking effects in dev.
  useEffect(() => {
    if (started.current) return;
    started.current = true;
    mutate({ url: window.location.origin, attempts: 1 });
  }, [mutate]);

  return <SetupFinishStatus finish={finish} />;
};
