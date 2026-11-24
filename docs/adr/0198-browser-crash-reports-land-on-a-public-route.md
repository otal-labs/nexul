# Browser crash reports land on a public route

A tab that freezes for good leaves nothing in the instance logs. The browser log relay and the long animation frame
report (`POST /api/logs`) both run on the page's main thread, so a page stuck in a loop never sends them, and the person
closes the tab. Finding the loop then took a person pausing the debugger on their own machine.

Decision: the web app asks Chrome for crash reports with JavaScript call stacks, and the server takes them on a public
route.

- **What asks for them.** Every response that serves the app's page carries
  `Document-Policy: include-js-call-stacks-in-crash-reports` and
  `Reporting-Endpoints: crash-reporting="/api/browser-reports"`. When the browser kills the tab, after **Exit page** on
  a frozen tab or when it runs out of memory, it uploads a `crash` report holding the reason and, for an unresponsive
  tab, the call stack it was stuck in.
- **Why the route is public.** The browser uploads the report itself, without the page's bearer token, so the route
  cannot check a session. It takes only `crash` reports, at most 20 to an upload of at most 64 KiB, under one rate limit
  of 1 a second with a burst of 20 shared by every caller, so a flood costs a few lines and never memory per caller.
  Lines go to `source=web` with no `user_id`, keep only the report URL's path, and are redacted like every browser line.
- **What it costs.** Anyone who can reach the instance can write crash lines, or use up the limit so real reports are
  dropped for a while. A line on this route is a hint to read with its user agent, never evidence of who saw it.

Firefox and Safari do not send crash reports, so a freeze there still leaves only the long animation frame report, and
only if the tab recovers.
