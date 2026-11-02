import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { HOVER_INTENT_MS, useHoverPrefetch } from "@/hooks/useHoverPrefetch";

const setup = () => {
  const prefetch = vi.fn();
  const client = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  const { result } = renderHook(() => useHoverPrefetch(prefetch), { wrapper });
  return { prefetch, handlers: () => result.current };
};

describe("useHoverPrefetch", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("prefetches nothing for a pointer that passes over the row", () => {
    const { prefetch, handlers } = setup();
    act(() => handlers().onPointerEnter());
    act(() => vi.advanceTimersByTime(HOVER_INTENT_MS - 1));
    act(() => handlers().onPointerLeave());
    act(() => vi.advanceTimersByTime(HOVER_INTENT_MS));
    expect(prefetch).not.toHaveBeenCalled();
  });

  it("prefetches once the pointer has rested on the row", () => {
    const { prefetch, handlers } = setup();
    act(() => handlers().onPointerEnter());
    act(() => vi.advanceTimersByTime(HOVER_INTENT_MS));
    expect(prefetch).toHaveBeenCalledTimes(1);
  });

  it("prefetches at once when the row takes focus", () => {
    const { prefetch, handlers } = setup();
    act(() => handlers().onFocus());
    expect(prefetch).toHaveBeenCalledTimes(1);
  });
});
