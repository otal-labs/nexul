import { autoPlayPlaces, type AutoPlayPlaces, type AutoPlayWhose } from "@nexul/client-core/autoPlayComputers";
import type { AutoPlayComputerRule } from "@nexul/client-core/autoPlayText";

import { useFetchMe } from "@/hooks/AuthHooks";
import { useListComputers } from "@/hooks/PairingHooks";
import type { Computer } from "@/models/Pairing";

// Where an auto play would run now, from the viewer's own computers; undefined until the viewer and their computers are read.
export const useAutoPlayPlaces = (autoPlay: AutoPlayWhose, rule: AutoPlayComputerRule | undefined): AutoPlayPlaces<Computer> | undefined => {
  const { data: me } = useFetchMe();
  const { data: computers } = useListComputers();
  if (!me || !computers) return undefined;
  return autoPlayPlaces(autoPlay, rule, me.user.id, computers.filter((c) => !c.unfinished));
};
