import { render, screen } from "@testing-library/react-native";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";

const classOf = (name: string) => String(screen.getByRole("button", { name }).props.className ?? "");

describe("Button", () => {
  it.each(["default", "destructive", "secondary", "outline"] as const)("draws a disabled %s button with no fill and no lower opacity", async (variant) => {
    await render(
      <Button variant={variant} disabled>
        <Text>Save</Text>
      </Button>,
    );
    expect(classOf("Save")).not.toMatch(/(^|\s)opacity-/);
    expect(classOf("Save")).not.toMatch(/(^|\s)bg-(brand|destructive-fill|secondary|card)(\/|\s|$)/);
    expect(screen.getByText("Save").props.className).toMatch(/text-muted-foreground/);
  });

  it("keeps the fill of a live button", async () => {
    await render(
      <Button>
        <Text>Save</Text>
      </Button>,
    );
    expect(classOf("Save")).toMatch(/(^|\s)bg-brand(\s|$)/);
  });
});
