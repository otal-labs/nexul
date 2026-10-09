import { RefreshControl, type RefreshControlProps } from "react-native";
import { useCSSVariable } from "uniwind";

// Pull to refresh is progress, so its spinner is the ember on the popover surface.
// Android's ScrollView clones this element and passes the list in as children, so every prop is forwarded.
export const RefreshList = (props: RefreshControlProps) => {
  const [brand, popover] = useCSSVariable(["--color-brand", "--color-popover"]);
  return <RefreshControl colors={[String(brand)]} tintColor={String(brand)} progressBackgroundColor={String(popover)} {...props} />;
};
