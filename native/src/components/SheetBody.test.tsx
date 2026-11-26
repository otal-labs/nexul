import { render, screen } from "@testing-library/react-native";
import { SafeAreaInsetsContext } from "react-native-safe-area-context";

import { SheetBody } from "@/components/SheetBody";
import { Text } from "@/components/ui/text";

describe("SheetBody", () => {
  it("clears the gesture bar by the bottom inset plus 16pt", async () => {
    await render(
      <SafeAreaInsetsContext.Provider value={{ top: 0, right: 0, bottom: 24, left: 0 }}>
        <SheetBody>
          <Text>Last control</Text>
        </SheetBody>
      </SafeAreaInsetsContext.Provider>,
    );
    const root = screen.toJSON();
    expect(root && !Array.isArray(root) && root.props.style).toMatchObject({ paddingBottom: 40 });
  });
});
