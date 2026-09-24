import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";

export const getMemorySkillKey = "getMemorySkill";

// The server holds the one copy of the nexul-memory skill, the same file the setup turns install.
export const useFetchMemorySkill = () =>
  useQuery({
    queryKey: [getMemorySkillKey],
    queryFn: async () => (await api.get<{ skill: string }>("/api/pairing/memory-skill")).data.skill,
    staleTime: Infinity,
  });
