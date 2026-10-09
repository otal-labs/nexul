import { View } from "react-native";

import { Microheader } from "@/components/Microheader";

// Each day opens with its label between two hairlines.
export const ChatDayDivider = ({ label }: { label: string }) => (
  <View accessible role="heading" className="flex-row items-center gap-3 px-5 pb-1 pt-5">
    <View className="h-px flex-1 bg-border" />
    <Microheader>{label}</Microheader>
    <View className="h-px flex-1 bg-border" />
  </View>
);
