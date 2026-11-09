import { act, render, screen } from "@testing-library/react-native";
import { Linking } from "react-native";

import { HtmlPageView, allowPageLoad } from "@/components/docs/HtmlPageView";

const post = (data: unknown) => screen.getByTestId("webview").props.onMessage({ nativeEvent: { data: JSON.stringify(data) } });

describe("HtmlPageView", () => {
  test("refuses a page navigating its top frame away, and lets its own frames load", () => {
    expect(allowPageLoad({ url: "https://example.com/phish", isTopFrame: true } as never)).toBe(false);
    expect(allowPageLoad({ url: "intent://open#Intent;end", isTopFrame: true } as never)).toBe(false);
    expect(allowPageLoad({ url: "about:blank", isTopFrame: true } as never)).toBe(true);
    expect(allowPageLoad({ url: "https://example.com/embed", isTopFrame: false } as never)).toBe(true);
  });

  test("ignores a message from anything but the page's bootstrap", async () => {
    await render(<HtmlPageView code="<p>hi</p>" />);
    await act(async () => post({ type: "height", height: 900 }));
    expect(screen.getByTestId("webview").props.style.height).toBe(40);
  });

  test("runs the block's code with the theme, fits the reported height, and opens a tapped link in the browser", async () => {
    const open = jest.spyOn(Linking, "openURL").mockResolvedValue(true);
    await render(<HtmlPageView code="<p>hello page</p>" />);
    const view = screen.getByTestId("webview");
    expect(view.props.source.html).toContain("<p>hello page</p>");
    expect(view.props.source.html).toContain("--font-sans:");

    await act(async () => post({ source: "nexul-html-page", type: "height", height: 321 }));
    expect(screen.getByTestId("webview").props.style.height).toBe(321);

    await act(async () => post({ source: "nexul-html-page", type: "link", url: "https://example.com/plan" }));
    expect(open).toHaveBeenCalledWith("https://example.com/plan");
  });
});
