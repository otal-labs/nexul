import { useMemo, useState } from "react";
import { Linking } from "react-native";
import { WebView, type WebViewMessageEvent } from "react-native-webview";
import type { ShouldStartLoadRequest } from "react-native-webview/lib/WebViewTypes";
import { useCSSVariable, useUniwind } from "uniwind";

import {
  HTML_PAGE_COLOR_TOKENS,
  HTML_PAGE_MIN_HEIGHT,
  htmlPageDocument,
  readHtmlPageMessage,
} from "@nexul/client-core/htmlPage";

// ponytail: system fonts stand in for the bundled ones, which a WebView can't reach without file access.
const FONTS = {
  "font-sans": "Inter, system-ui, sans-serif",
  "font-mono": "'JetBrains Mono', ui-monospace, monospace",
  "font-display": "Fraunces, Georgia, serif",
};

const COLOR_VARIABLES = HTML_PAGE_COLOR_TOKENS.map((token) => `--color-${token}`);

// The page itself loads, and frames inside it; a page that tries to navigate away goes nowhere.
export const allowPageLoad = ({ url, isTopFrame }: ShouldStartLoadRequest) =>
  isTopFrame === false || url === "about:blank" || url.startsWith("data:");

// One object per page and theme: a new source object reloads the WebView.
const useHtmlPageSource = (code: string) => {
  const appearance = useUniwind().theme === "light" ? "light" : "dark";
  const colors = useCSSVariable(COLOR_VARIABLES);
  const key = colors.join("|");
  return useMemo(() => {
    const variables: Record<string, string> = { ...FONTS, radius: "8px" };
    HTML_PAGE_COLOR_TOKENS.forEach((token, i) => {
      const value = colors[i];
      if (value !== undefined) variables[token] = String(value);
    });
    return { html: htmlPageDocument(code, { appearance, variables }) };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- key stands for colors, a new array every render
  }, [code, appearance, key]);
};

// A page from a doc body, sized to its content; it never reaches the app's session, which lives outside the WebView.
export const HtmlPageView = ({ code }: { code: string }) => {
  const source = useHtmlPageSource(code);
  const [height, setHeight] = useState(HTML_PAGE_MIN_HEIGHT);

  const onMessage = (event: WebViewMessageEvent) => {
    const message = readHtmlPageMessage(event.nativeEvent.data);
    if (message?.type === "height") setHeight(message.height);
    if (message?.type === "link") void Linking.openURL(message.url);
  };

  return (
    <WebView
      source={source}
      originWhitelist={["*"]}
      onShouldStartLoadWithRequest={allowPageLoad}
      onMessage={onMessage}
      setSupportMultipleWindows={false}
      javaScriptCanOpenWindowsAutomatically={false}
      mixedContentMode="never"
      scrollEnabled={false}
      nestedScrollEnabled={false}
      overScrollMode="never"
      // A height the page reports and a see-through canvas are values className can't carry onto a third-party view.
      style={{ height, backgroundColor: "transparent" }}
      accessibilityLabel="HTML page"
    />
  );
};
