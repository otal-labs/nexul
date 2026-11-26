import { useSearchParams } from "react-router";

export type DeployFilter = "all" | "failed";

// The deploy list's All / Failed, held in the query so the tab row's toggle and the list read one value.
export const useDeployFilter = (): [DeployFilter, (filter: DeployFilter) => void] => {
  const [params, setParams] = useSearchParams();
  const filter: DeployFilter = params.get("show") === "failed" ? "failed" : "all";
  const setFilter = (next: DeployFilter) =>
    setParams(
      (current) => {
        const updated = new URLSearchParams(current);
        if (next === "all") updated.delete("show");
        if (next === "failed") updated.set("show", "failed");
        return updated;
      },
      { replace: true },
    );
  return [filter, setFilter];
};
