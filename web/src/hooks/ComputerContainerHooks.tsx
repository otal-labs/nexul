import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { ComputerContainer } from "@/models/ComputerContainer";

export const getComputerContainersKey = "getComputerContainers";

// Docker pushes nothing to Nexul, so the list is read when the details open and on a refetch.
export const useFetchComputerContainers = (computerId: string) =>
  useQuery({
    queryKey: [getComputerContainersKey, computerId],
    queryFn: () =>
      api
        .get<{ containers: ComputerContainer[] }>(`/api/pairing/computers/${computerId}/containers`)
        .then((r) => r.data.containers),
    enabled: !!computerId,
  });
