import { useFetchMe } from "@/hooks/AuthHooks";
import { useActionState } from "@/hooks/useActionState";
import type { Trail } from "@/models/Trail";

// The run's starter, or a plays:write holder, may answer, stop or continue it; anyone else sees the run read-only.
export const useCanActOnTrail = (trail: Trail): boolean => {
  const { data: me } = useFetchMe();
  const canWrite = useActionState({ permission: "plays:write", projectId: trail.project_id }).state === "live";
  return trail.starter_id === me?.user.id || canWrite;
};
