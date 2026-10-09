import { act, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("shaders/react", () => {
  const Layer = () => null;
  return {
    Shader: () => <div data-testid="live-field" />,
    FilmGrain: Layer,
    FlowField: Layer,
    MultiPointGradient: Layer,
    SolidColor: Layer,
  };
});

const renderField = async ({ reduced, gpu, live = true }: { reduced: boolean; gpu: boolean; live?: boolean }) => {
  vi.resetModules();
  vi.stubGlobal("matchMedia", (query: string) => ({
    matches: reduced && query.includes("reduce"),
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
  }));
  if (gpu) vi.stubGlobal("navigator", { ...navigator, gpu: {} });
  const { ShowcaseField } = await import("@/components/showcase/ShowcaseField");
  // Loads the lazy live layer up front so act() settles it; a missing guard then shows on the first assertion.
  await import("@/components/showcase/LiveLightField");
  await act(async () => {
    render(<ShowcaseField live={live} />);
  });
};

afterEach(() => vi.unstubAllGlobals());

describe("ShowcaseField", () => {
  it("mounts the live field when WebGPU is there and motion is allowed", async () => {
    await renderField({ reduced: false, gpu: true });
    expect(screen.getByTestId("live-field")).toBeInTheDocument();
  });

  it("keeps to the still unless the caller asks for the live field", async () => {
    await renderField({ reduced: false, gpu: true, live: false });
    expect(screen.queryByTestId("live-field")).not.toBeInTheDocument();
  });

  it("keeps to the still under reduced motion", async () => {
    await renderField({ reduced: true, gpu: true });
    expect(screen.queryByTestId("live-field")).not.toBeInTheDocument();
  });

  it("keeps to the still without WebGPU", async () => {
    await renderField({ reduced: false, gpu: false });
    expect(screen.queryByTestId("live-field")).not.toBeInTheDocument();
  });
});
