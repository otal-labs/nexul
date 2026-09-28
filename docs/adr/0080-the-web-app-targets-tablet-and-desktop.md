# The web app targets tablet and desktop; phones get the Android app

The web app was held to a mobile-first rule: every page built at 320, 375,
and 414px first and verified at those widths before it counted as done. A
self-hosted dev platform's boards, canvases, editors, and settings are tablet
and desktop work, so that rule spent verification effort on every change for
a layout nobody chose to use, and still produced a cramped phone experience.

Decision: `web/` is built at 768px first and verified at 768, 1024, and
1440px. Phones are served by the Android app in `native/`, which carries the
pages worth having on a phone, laid out for one. Below 768px the web app
shows a dismissible banner instead of blocking: on Android it links to the
latest app release, elsewhere it says the web app is built for tablet and
desktop. Dismissal is remembered per device.

Existing small-screen classes stay where they are. Removing them costs work
and risks tablet layouts; new work simply stops building and verifying below
768px.

`website/`, the marketing and docs site, keeps the mobile-first rule, because
people do read landing pages and documentation on phones.

Rejected: blocking the web app below 768px, which stops someone who only has
a phone from doing anything at all when the page would mostly still work.

Decided 2026-09-28.
