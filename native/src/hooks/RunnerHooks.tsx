import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { Runner } from "@/models/Runner";

export const getRunnersKey = "getRunners";

export const useFetchRunners = () =>
  useQuery({
    queryKey: [getRunnersKey],
    queryFn: () => api.get<Runner[]>("/api/runners"),
  });
