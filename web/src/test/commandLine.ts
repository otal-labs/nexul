// A CommandBlock line is split into one span per argument, so it is found by the whole line's text.
export const commandLine = (pattern: string | RegExp) => (_: string, element: Element | null) =>
  element?.tagName === "CODE" && (typeof pattern === "string" ? element.textContent?.endsWith(pattern) === true : pattern.test(element.textContent ?? ""));
