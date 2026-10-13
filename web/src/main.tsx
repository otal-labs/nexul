import "@fontsource-variable/fraunces/full.css";
import "@fontsource-variable/inter";
import "@fontsource-variable/jetbrains-mono";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { createBrowserRouter, RouterProvider } from "react-router";
import { TeamProgressProto } from "./proto/team-progress/TeamProgressProto";
import { installBrowserLogs } from "./lib/browserLogs";
import "./lib/motion";
import "./index.css";

installBrowserLogs();

// proto:team-progress (throwaway prototype mount, not for master)
const protoRouter = createBrowserRouter([{ path: "/proto/team-progress", element: <TeamProgressProto /> }]);
const isProto = window.location.pathname.startsWith("/proto/");
// end proto:team-progress

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    {isProto && <RouterProvider router={protoRouter} />}
    {!isProto && <App />}
  </StrictMode>,
);
