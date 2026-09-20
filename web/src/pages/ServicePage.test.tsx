import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { describe, expect, it } from "vitest";

import { ServicePage } from "@/pages/ServicePage";

const StackPageStub = () => {
  const { stackId } = useParams();
  return <p>stack page {stackId}</p>;
};

describe("ServicePage", () => {
  it("redirects /services/:id to /stacks/:id, the same id", () => {
    render(
      <MemoryRouter initialEntries={["/services/svc-1"]}>
        <Routes>
          <Route path="/services/:serviceId" element={<ServicePage />} />
          <Route path="/stacks/:stackId" element={<StackPageStub />} />
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByText("stack page svc-1")).toBeInTheDocument();
  });
});
