import { View } from "react-native";

import type { TrailState } from "@nexul/client-core/trail";

import { cn } from "@/lib/utils";
import { trailStateDot } from "@/models/Trail";

// A run's state as the web's hue, the one colour on its row.
export const RunStateDot = ({ state }: { state: TrailState }) => <View className={cn("size-2 rounded-full", trailStateDot(state))} />;
