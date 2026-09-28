# React Native app: stack recommendation

Researched 2026-09-28 against official docs, changelogs, source repositories and
the npm registry. Every version below is the npm `latest` tag on that date
unless a date says otherwise. Facts about pricing and store policy are as
published on that date and will drift.

## Recommended stack

| Area | Choice | Why, in one line | Source |
|---|---|---|---|
| Framework | Expo SDK 57 (RN 0.86, React 19.2), move to SDK 58 when it leaves beta | Stable today; SDK 58 (RN 0.88) has been in beta since 2026-09-15 and ships a few weeks after RN 0.88 | https://expo.dev/changelog/sdk-57, https://expo.dev/changelog/sdk-58-beta |
| Architecture | New Architecture (the only option), Hermes v1, React Compiler on | Legacy arch ended with SDK 54; Hermes v1 default since SDK 56; compiler on in the default template since SDK 54 | https://expo.dev/changelog/sdk-55, https://expo.dev/changelog/sdk-56, https://expo.dev/changelog/sdk-54 |
| Navigation | Expo Router | File routes like the web app; no longer depends on React Navigation since SDK 56; native tabs and toolbars stable in SDK 58 | https://expo.dev/changelog/sdk-56, https://expo.dev/changelog/sdk-58-beta |
| OTA client | `expo-updates`, `runtimeVersion: { policy: "fingerprint" }` | Built-in crash fallback, rollback directive, bytecode diffing on by default | https://docs.expo.dev/eas-update/runtime-versions/, https://docs.expo.dev/eas-update/error-recovery/ |
| OTA server | One central update server owned by the project (EAS Update to start), never per-instance | Per-instance URLs need anti-bricking disabled, which Expo says not to ship; a header override to pick a channel is production-safe | https://docs.expo.dev/eas-update/override/ |
| Version skew | App reads the instance version, API stays additive, channel per server line only when a breaking change forces it | Cheapest path; the header override keeps the escape hatch | https://docs.expo.dev/eas-update/override/ |
| Styling | Uniwind (Tailwind v4, CSS variables, runtime `updateCSSVariables`) | Same `@theme` tokens as `web/src/index.css`; palettes switch at runtime like web; NativeWind v5 is still RC | https://docs.uniwind.dev/theming/update-css-variables, https://github.com/nativewind/nativewind/discussions/1818 |
| Components | react-native-reusables, Uniwind flavour | shadcn/ui port, installed through the shadcn CLI | https://github.com/founded-labs/react-native-reusables |
| Auth | `expo-web-browser` `openAuthSessionAsync` + custom scheme + one-time code exchange | Works for any instance host; universal links cannot vary per instance | https://docs.expo.dev/versions/latest/sdk/webbrowser/, https://docs.expo.dev/linking/ios-universal-links/ |
| Token storage | `expo-secure-store` (sync `getItem` at boot), `expo-local-authentication` for an optional lock | Keychain/Keystore; small values only | https://docs.expo.dev/versions/latest/sdk/securestore/ |
| Other persisted state | zustand `persist` on `expo-sqlite/kv-store` (sync API) or `react-native-mmkv` v4 | Synchronous storage means no hydration flash | https://docs.expo.dev/versions/latest/sdk/sqlite/, https://github.com/pmndrs/zustand/blob/main/docs/reference/integrations/persisting-store-data.md |
| Server state | TanStack Query v5 + `focusManager`/`onlineManager` + async-storage persister | The documented RN setup | https://tanstack.com/query/latest/docs/framework/react/react-native |
| Lists | FlashList v2 by default, Legend List v3 for the chat thread | FlashList is the default; Legend List has chat anchoring and keyboard helpers built in | https://shopify.github.io/flash-list/docs/v2-changes/, https://legendapp.com/open-source/list/v3/overview/ |
| Motion, gestures, sheets | Reanimated 4, Gesture Handler (Expo-pinned), Expo Router `formSheet` for sheets | All ship in the Expo template; native sheets without an extra library | https://docs.expo.dev/router/advanced/modals/ |
| Markdown | `react-native-enriched-markdown` | Native text, GFM, streaming; no WebView | https://github.com/software-mansion-labs/react-native-enriched-markdown |
| Doc editing | WebView of the instance's own editor route (or TenTap); not Expo DOM components | Collaborative Tiptap + Yjs is already on web; DOM components cannot be OTA-updated | https://docs.expo.dev/guides/dom-components/ |
| Push | Expo Push Service called directly by each instance, id-only payloads | Publisher-owned APNs/FCM credentials live in one Expo project; tokens work from any server | https://docs.expo.dev/push-notifications/sending-notifications/, https://docs.expo.dev/push-notifications/faq/ |
| Code sharing | `@nexul/sdk` + React-free `web/src/models` through Metro `watchFolders`; no shared hooks yet | 53 of 56 model files import only `zod`; hooks pull in axios, stores and React | https://metrobundler.dev/docs/configuration/ |
| Tests | jest-expo + React Native Testing Library; Maestro for E2E | Expo's documented setup; Maestro is Apache-2.0 and runs locally or in EAS Workflows | https://docs.expo.dev/develop/unit-testing/, https://docs.expo.dev/eas/workflows/examples/e2e-tests/ |
| Builds and store | EAS Build free tier + EAS Submit; local builds as fallback | 15 iOS + 15 Android builds a month free | https://expo.dev/pricing |

---

## 1. Framework baseline

**Findings**

- Current stable is Expo SDK 57, released 2026-06-30, on React Native 0.86 and
  React 19.2 (npm: `expo` 57.0.25, 2026-09-24). The SDK 57 default template pins
  `react-native` 0.86.3, `react` 19.2.3, `expo-router` ~57.0.23,
  `react-native-reanimated` 4.5.1, `react-native-gesture-handler` ~2.32.0,
  `typescript` ~6.0.3 and turns on `experiments.reactCompiler` and
  `experiments.typedRoutes`.
  Sources: https://expo.dev/changelog/sdk-57,
  https://github.com/expo/expo/blob/sdk-57/templates/expo-template-default/package.json,
  https://github.com/expo/expo/blob/sdk-57/templates/expo-template-default/app.json
- SDK 58 beta was published 2026-09-15 on React Native 0.88 RC. The beta "will
  last three to four weeks" and SDK 58 ships "shortly after" RN 0.88 stable.
  Breaking: iOS needs the UIKit scene-based lifecycle for iOS 27, R8 is on by
  default in Android release builds, `File.write()` becomes async, foreground
  notifications show by default. The `main` branch template already uses React
  19.3.0 and RN 0.88.0-rc.2, matching the web app's React 19.3.
  Sources: https://expo.dev/changelog/sdk-58-beta,
  https://github.com/expo/expo/blob/main/templates/expo-template-default/package.json
- New Architecture: "SDK 54 is the final release to include Legacy Architecture
  support", and `newArchEnabled` was removed from app config in SDK 55
  (2026-02-25). There is nothing to decide here.
  Sources: https://expo.dev/changelog/sdk-54, https://expo.dev/changelog/sdk-55
- Hermes v1 became the default engine in SDK 56 (2026-05-21). SDK 57 fixed a
  memory regression that hit apps using Reanimated/worklets in `expo@57.0.9`.
  SDK 56 also raised the minimum to iOS 16.4 and Xcode 26.4, and Expo Go for
  SDK 56+ is not in the stores (TestFlight or CLI install only).
  Sources: https://expo.dev/changelog/sdk-56, https://expo.dev/changelog/sdk-57
- React Compiler: "React Compiler is now enabled in the default template. We
  recommend using it in your projects" (SDK 54, 2025-09-10). It is still an
  `experiments.reactCompiler` flag in app config.
  Sources: https://expo.dev/changelog/sdk-54, https://docs.expo.dev/guides/react-compiler/
- Expo Router vs React Navigation: since SDK 56, "expo-router no longer depends
  on react-navigation" (codemods provided). SDK 58 makes data loaders, SSR,
  middleware, native tabs, toolbars and the navigation integration stable, and
  reworks the navigation core, with some APIs moved or removed.
  Sources: https://expo.dev/changelog/sdk-56, https://expo.dev/changelog/sdk-58-beta
- Bare React Native is not worth it: every recommendation below (updates,
  secure store, notifications, auth session, fingerprinting) is an Expo module,
  and Expo supports bun as a package manager
  (`bun create expo-app --template default@sdk-55`).
  Source: https://expo.dev/changelog/sdk-55

**Recommendation.** Expo, Continuous Native Generation (no committed `ios/`
and `android/`), Expo Router, React Compiler on, TypeScript strict. Start on
SDK 57 and upgrade to 58 in its first stable week, before there are users to
break. SDK 58's Router rework is the one migration you want to take before
writing many screens.

**Open questions for the owner**

- Start on SDK 58 beta directly (React 19.3 like web, no Router migration
  later) and accept beta churn, or on SDK 57 stable?
- Minimum OS: SDK 56+ already means iOS 16.4. Any reason to go higher?

## 2. OTA updates

**Findings: store policy**

- Apple Developer Program License Agreement, section 3.3.1(B) "Executable
  Code": "Interpreted code may be downloaded to an Application but only so long
  as such code: (a) does not change the primary purpose of the Application ...
  (b) does not bypass signing, sandbox, or other security features of the OS;
  and (c) for Applications distributed on the App Store, does not create a
  store or storefront for other Applications." (The clause the ecosystem calls
  "3.3.2" is now numbered 3.3.1(B); 3.3.2 is Regulatory Compliance.)
  Source: https://developer.apple.com/support/terms/apple-developer-program-license-agreement/
- App Review Guideline 2.5.2: apps may not "download, install, or execute code
  which introduces or changes features or functionality of the app". Guideline
  4.7 covers JavaScript that is not embedded in the binary and says "You are
  responsible for all such software offered in your app".
  Source: https://developer.apple.com/app-store/review/guidelines/
- Google Play Device and Network Abuse policy: an app "may not download
  executable code (such as dex, JAR, .so files) from a source other than Google
  Play. This restriction does not apply to code that runs in a virtual machine
  or an interpreter", but interpreted code loaded at run time "must not allow
  potential violations of Google Play policies".
  Source: https://support.google.com/googleplay/android-developer/answer/16273414

Result: JS-only fixes and features within the app's advertised purpose are
allowed on both stores. Native changes need a store release, and the
fingerprint policy enforces that automatically.

**Findings: expo-updates**

- Protocol: open spec. Requests carry `expo-protocol-version`, `expo-platform`,
  `expo-runtime-version`; the server answers with a manifest or a directive.
  The `rollBackToEmbedded` directive tells the client to use the embedded
  bundle. Manifests and directives can be code-signed; the client verifies them
  against an embedded certificate. Source: https://docs.expo.dev/technical-specs/expo-updates-1/
- Runtime version: "If you want to make incompatible updates extremely
  unlikely, at the cost of making it necessary to create builds more often,
  then you can use the `fingerprint` policy." `appVersion` fails silently if you
  forget to bump the version after a native change.
  Source: https://docs.expo.dev/eas-update/runtime-versions/
- Crash safety: an update that crashes on launch is "marked as 'failed'
  locally and will not be launched again", and the app falls back to the most
  recent update that launched successfully.
  Source: https://docs.expo.dev/eas-update/error-recovery/
- Rollback: `eas update:rollback` either republishes an earlier update or
  instructs clients to run the embedded one.
  Source: https://docs.expo.dev/eas-update/rollbacks/
- Bundle size: Hermes bytecode diffing arrived in SDK 55 (opt-in, about 75%
  smaller downloads) and is on by default from SDK 56 (diffs average 58% of the
  full bundle). Sources: https://expo.dev/changelog/sdk-55, https://expo.dev/changelog/sdk-56

**Findings: changing where updates come from at runtime**

- `Updates.setUpdateURLAndRequestHeadersOverride({ updateUrl, requestHeaders })`
  exists (SDK 52+) but "requires disableAntiBrickingMeasures to be set to true",
  takes effect only after the app is closed and reopened, and the docs say "Do
  not enable this in your production builds". With anti-bricking off, a
  crashing update cannot be recovered and "The user would need to uninstall
  and reinstall the app."
  Sources: https://docs.expo.dev/versions/latest/sdk/updates/, https://docs.expo.dev/eas-update/override/
- `Updates.setUpdateRequestHeadersOverride(headers)` (SDK 54+, `expo-updates`
  0.29+) does not need anti-bricking disabled. Headers must be declared in
  `updates.requestHeaders` at build time, and `expo-channel-name` can be
  overridden. Follow it with `fetchUpdateAsync()` + `reloadAsync()`, or wait for
  the next launch. Source: https://docs.expo.dev/eas-update/override/
- hot-updater (0.36.15, 1.0 in RC) accepts `baseURL` as an async function
  "called before each update check", so a per-instance URL is supported there,
  and it has RSA-SHA256 bundle signing, `.bsdiff` patches and automatic
  rollback on failed first launch. It replaces `expo-updates` rather than
  building on it. Sources: https://github.com/gronxb/hot-updater (docs
  `react-native-api/init.mdx`, `guides/bundle-signing.mdx`), https://hot-updater.dev/docs/get-started/introduction

**Findings: hosting and price**

- EAS Update: Free covers 1,000 monthly active updaters (hard cap, no overage);
  Starter $19/month covers 3,000 and then $0.005 per MAU; Production
  $199/month covers 50,000. There is no listed open-source discount.
  Source: https://expo.dev/pricing
- EAS Update code signing "is only available to accounts subscribed to the EAS
  Production or Enterprise plans." Source: https://docs.expo.dev/eas-update/code-signing/
- Self-hosted options that speak the same protocol: Expo's reference server
  (explicitly "not guaranteed to be complete, stable, or performant enough",
  https://github.com/expo/custom-expo-updates-server) and Xprem, formerly
  expo-open-ota, an MIT Go server with S3/CDN storage, rollouts and rollbacks
  (https://github.com/mercuretechnologies/expo-open-ota).
- CodePush successors: App Center (and hosted CodePush) retired 2025-03-31.
  Revopush keeps the CodePush API and is priced at a free tier up to 1,000 MAU,
  then $500/month for up to 1M MAU. Neither adds anything over `expo-updates` in
  an Expo app. Sources: https://github.com/revopush/react-native-code-push,
  https://blog.codemagic.io/react-native-ota-tools-in-2026/ (secondary, for pricing)

**The per-instance question**

Could each Nexul instance serve the JS bundle matching its own server version?
It is technically possible, but it is the wrong design:

1. With `expo-updates` it needs the URL override, which requires disabling
   anti-bricking, which Expo says must not ship to production.
2. The publisher answers to Apple for every byte the app runs (guideline 4.7).
   An instance that serves JS turns every self-hoster into a code distributor
   under the publisher's signature. Without code signing, a compromised or
   modified instance can ship arbitrary code to its users' phones.
3. A user connected to two instances on different versions would need the app
   to reload (and on `expo-updates`, relaunch) on every switch.
4. Every server release would need a mobile bundle build and signing step
   inside the instance's install and upgrade path.

The same result is available without those costs. Keep one central update
server. The app reads the connected instance's version from the API and sets
`expo-channel-name` with `setUpdateRequestHeadersOverride` to a channel per
server release line (for example `server-0.2`). The update server, not the
instance, picks the bundle, and the header override is production-safe.

**Recommendation.** `expo-updates` with the fingerprint policy and
`checkAutomatically: ON_LOAD`. One channel (`production`) at first. Handle
version skew in the app: call the instance's version endpoint on connect,
keep a min/max supported server range in the app, gate features on reported
capabilities, and show "update the app" or "your instance is too old" when out
of range. Keep the HTTP API additive. Add per-server-line channels through the
header override only when a breaking server change actually happens. Start on
EAS Update Free. Move to a self-hosted protocol server (Xprem, or the spec
implemented in Go on the project's own infrastructure) when MAU nears 1,000 or
when bundle signing is needed. `updates.url` is baked into the binary, so the
switch rides on a store release.

**Open questions for the owner**

- Does the Free-tier 1,000 MAU cap need to be solved before launch? It depends
  on expected installs, and self-hosting from day one avoids a later store
  release.
- Is code signing a requirement from day one? On EAS it costs $199/month; self-hosted
  it is free.
- Should the server expose an explicit capability list next to its version, so
  the app gates on features instead of version numbers?

## 3. Styling and components

**Findings**

- NativeWind: stable `nativewind` 4.2.7 is on Tailwind v3. v5 (Tailwind v4)
  published `5.0.0-rc.0` on 2026-09-13, with performance work and migration
  docs still open, and a known Android animation-cancellation limitation
  tracked upstream in Reanimated.
  Source: https://github.com/nativewind/nativewind/discussions/1818 (npm tags checked 2026-09-28)
- Uniwind (`uniwind` 1.12.0, 2026-09-04, MIT, by the Unistyles team) is built
  for Tailwind v4: CSS-first config, `@theme`, CSS variables, build-time
  compilation. Themes are `@variant` blocks in `global.css` registered in
  `metro.config.js`, with no limit on count. "Every theme must define the same
  CSS variables", and switching uses `Uniwind.setTheme(name)`.
  `Uniwind.updateCSSVariables(theme, vars)` changes a theme's variables at
  runtime and persists per theme. A paid Pro tier adds a C++ ShadowTree engine
  and native theme transitions. The free tier is the production baseline.
  Sources: https://uniwind.dev/, https://docs.uniwind.dev/theming/custom-themes,
  https://docs.uniwind.dev/theming/update-css-variables
- Unistyles 3 (3.3.0, 2026-07-10) themes are plain JS objects with adaptive
  light/dark and runtime updates. It fits well, but it is a StyleSheet API
  rather than `className`, so web's Tailwind tokens would be restated as JS.
  Source: https://unistyl.es/v3/guides/theming/
- Tamagui (2.7.7, 2026-08-15) is its own token system and compiler, the
  furthest from a Tailwind v4 + shadcn web app. (npm registry)
- react-native-reusables (8.7k stars) is the shadcn/ui port. Its CLI
  (`@react-native-reusables/cli` 0.7.1) "Uses the shadcn CLI under the hood".
  It supports `--styling-library nativewind|uniwind`, and registry items live at
  `https://reactnativereusables.com/r/<uniwind|nativewind>/<component>.json`,
  so `shadcn add <url>` works too.
  Source: https://github.com/founded-labs/react-native-reusables (docs `cli.mdx`, `create-your-own-registry.mdx`)
- Fit with this repo: web palettes are TS objects in
  `web/src/lib/themePalettes.ts` (per-role overrides for light and dark, twelve
  palettes, some values in `oklch(...)`), applied at runtime over the defaults in
  `web/src/index.css`. This matches Uniwind's model: define `light`/`dark` in
  `global.css` from the same token names, then call
  `Uniwind.updateCSSVariables` with the chosen palette's roles.

**Recommendation.** Uniwind + react-native-reusables (Uniwind flavour).
Mirror `index.css` token names in `native/global.css`. Apply palettes with
`updateCSSVariables` fed from the same palette data web uses. Revisit
NativeWind only if v5 goes stable and Uniwind stalls.

**Open questions for the owner**

- Is it acceptable to run a converter over palette colours (oklch to hex) at
  build or load time? Uniwind converts `oklch` in CSS at build time, but none of
  the sources say whether runtime `updateCSSVariables` accepts `oklch()` strings.
  Test this first.
- Mono Console fonts (Inter, JetBrains Mono) need to be bundled via
  `expo-font`. Confirm that's wanted rather than system fonts on mobile.

## 4. Auth against a per-instance URL

**Findings**

- `WebBrowser.openAuthSessionAsync(url, redirectUrl)` "Opens the url with
  Safari in a modal using ASWebAuthenticationSession" on iOS and uses Custom
  Tabs on Android. The redirect must use the app's scheme ("demo:// not
  https://"). `preferEphemeralSession` stops cookie sharing with Safari.
  Source: https://docs.expo.dev/versions/latest/sdk/webbrowser/
- Universal links are fixed at build time: domains go into
  `ios.associatedDomains` in the entitlements, and each domain must host
  `apple-app-site-association`. A store build cannot claim every customer's
  domain. Source: https://docs.expo.dev/linking/ios-universal-links/
- RFC 8252: private-use schemes must be reverse-domain (§7.1). Public native
  clients "MUST implement" PKCE (§8.1), and apps "MUST NOT use embedded
  user-agents" (§8.12). Source: https://datatracker.ietf.org/doc/html/rfc8252
- Today the instance's OAuth callback redirects to
  `<spa>/login?token=<session token>` (`internal/auth/handler.go`, `callbackGET`).
  GitHub, Google and Discord only ever see the instance's own https callback,
  which is why a custom scheme works without registering anything new with them.
- `expo-secure-store` uses Keychain (iOS) and Keystore-backed storage (Android).
  Large values "can be rejected ... Historically, some iOS releases refused
  values above roughly 2048 bytes". There are synchronous `getItem`/`setItem`.
  `requireAuthentication` ties an item to biometrics, but items become
  unreadable when biometric enrolment changes. iOS keychain items survive an
  uninstall and Android's do not. Source: https://docs.expo.dev/versions/latest/sdk/securestore/
- MMKV v4 (`react-native-mmkv` 4.3.2) supports AES-128/256 encryption with a
  caller-supplied key, so the key itself would still live in the keychain.
  Source: https://github.com/mrousavy/react-native-mmkv
- `expo-local-authentication` `authenticateAsync` needs
  `NSFaceIDUsageDescription`, and Face ID does not work in Expo Go.
  Source: https://docs.expo.dev/versions/latest/sdk/local-authentication/

**Recommendation.** Instance-first sign-in: the user pastes an instance URL or
scans the existing connection token (QR), and the app fetches the instance's
enabled providers. For each provider, open
`openAuthSessionAsync("https://<instance>/api/auth/<provider>/start?client=mobile&code_challenge=...", "io.nexul.app://auth")`.
The instance does the provider dance as it does for web, then redirects to
`io.nexul.app://auth?code=<one-time code>` instead of putting a session token
in the URL. The app exchanges code + PKCE verifier for the bearer token. Store
the token per instance in SecureStore (key = instance id). Keep non-secret state
(the instance list, the active instance, the theme) in the zustand persist
store. Make biometric unlock an app-level gate (`authenticateAsync` on resume),
not `requireAuthentication` on the token, so a new fingerprint does not
silently log the user out.

**Open questions for the owner**

- Server change needed: a `client=mobile` branch of the OAuth start/callback
  with PKCE and a one-time code endpoint. The custom scheme can be claimed by
  any app, so the token must never travel in the redirect.
- Multiple instances signed in at once (like the desktop app), or one at a time?
- Scheme name (`io.nexul.app` or `nexul`)? RFC 8252 wants the reverse-domain
  form.

## 5. Data: TanStack Query, zustand, WebSocket

**Findings**

- TanStack Query's RN guide covers four things: `onlineManager` with
  `expo-network` or NetInfo; `focusManager` on the AppState `change` event; a
  `useFocusEffect` hook to refetch stale queries on screen focus; and
  `subscribed: false` to park queries on unfocused screens.
  Source: https://tanstack.com/query/latest/docs/framework/react/react-native
- Persistence: `createAsyncStoragePersister` takes any
  `getItem/setItem/removeItem` storage, throttles writes by default
  (1000 ms), and is used through `PersistQueryClientProvider`, where `gcTime`
  bounds retention. Source: https://tanstack.com/query/latest/docs/framework/react/plugins/createAsyncStoragePersister
- zustand `persist` accepts any storage via `createJSONStorage`. Async storage
  hydrates in a microtask, so the first render sees defaults ("your app might
  think the user is not logged in"). Synchronous storage hydrates at creation.
  Source: https://github.com/pmndrs/zustand/blob/main/docs/reference/integrations/persisting-store-data.md
- Synchronous storage options: `expo-sqlite/kv-store` (an AsyncStorage drop-in
  with `getItemSync`/`setItemSync`), `expo-sqlite/localStorage/install` (a
  `localStorage` polyfill, a no-op on web), or `react-native-mmkv` v4 (Nitro
  module, RN 0.76+). Sources: https://docs.expo.dev/versions/latest/sdk/sqlite/,
  https://github.com/mrousavy/react-native-mmkv
- AppState reports `active`, `background` and `inactive` (iOS only).
  Source: https://reactnative.dev/docs/appstate. iOS suspends backgrounded apps
  and open sockets do not survive that. This is platform behaviour, not a
  documented RN guarantee, so design for it.

**Recommendation.** Import the `localStorage` polyfill from `expo-sqlite`, and
existing web zustand stores that use `createJSONStorage(() => localStorage)`
work unchanged and hydrate synchronously. That is one less native dependency
than MMKV, and MMKV stays an option if profiling says so. TanStack Query with
the four RN integrations plus the async-storage persister on
`expo-sqlite/kv-store`, `gcTime` about 24 h. WebSocket: close on `background`,
reconnect with backoff on `active`, then invalidate queries whose topics may
have changed while away. The socket carries no catch-up guarantee, so the
refetch is the catch-up. Pushes cover the time the app is closed.

**Open questions for the owner**

- Does the live WebSocket have a resume cursor (last event id)? If yes, the
  app can replay instead of refetching everything on resume.

## 6. Lists, motion, sheets

**Findings**

- FlashList v2 (2.3.2, 2026-06-10) is New-Architecture-first. It needs no size
  estimates, adds `masonry`, `onStartReached` and `maintainVisibleContentPosition`
  with `startRenderingFromBottom` for chat, and drops `inverted`. An open
  issue reports scroll-direction confusion when paginating chat on v2.
  Sources: https://shopify.github.io/flash-list/docs/v2-changes/,
  https://github.com/Shopify/flash-list/issues/1844
- Legend List (`@legendapp/list` 3.4.0, 2026-09-21; the docs header still says
  beta) offers `initialScrollAtEnd`, `maintainScrollAtEnd`,
  `maintainVisibleContentPosition`, and `KeyboardAwareLegendList` with
  `useKeyboardChatComposerInset`. It also runs on web without React Native.
  Source: https://legendapp.com/open-source/list/v3/overview/
- Motion: Reanimated 4.7.0 (2026-09-18) and Gesture Handler 3.3.0
  (2026-09-11; 2.x kept under the `legacy` tag). Expo pins them per SDK (SDK
  57: Reanimated 4.5, RNGH 2.32; SDK 58 template: Reanimated 4.7, RNGH 3.2).
  Sources: npm registry, SDK templates above
- Sheets: Expo Router `presentation: 'formSheet'` gives native sheets with
  detents (`sheetAllowedDetents`, `fitToContents`, grabber). Android allows at
  most 3 detents and no nested navigators inside. Source:
  https://docs.expo.dev/router/advanced/modals/. Library alternatives:
  `@gorhom/bottom-sheet` 5.2.14 (last release 2026-05-09) and
  `@lodev09/react-native-true-sheet` 3.11.16 (native, 2026-09-27).

**Recommendation.** FlashList v2 for boards, tickets, logs and feeds. Legend
List only for the chat thread (anchoring plus keyboard composer inset). Always
install versions with `npx expo install` so the SDK pins win. Sheets as
`formSheet` routes, and add a sheet library only when a sheet must not be a
route.

## 7. Rich text and markdown

**Findings**

- Expo DOM components (`'use dom'`) run web React in a WebView behind a proxy.
  Props are serializable and async; native actions are async functions; there
  are no `children` or native views; startup is slower than Hermes bytecode.
  Critically: "DOM components can currently only be embedded and do not
  support OTA updates." Source: https://docs.expo.dev/guides/dom-components/
- TenTap (`@10play/tentap-editor` 1.0.1, last release 2025-11-27, repo pushed
  2026-07-27, 46 open issues) is Tiptap 3 plus ProseMirror in a WebView. Custom
  Tiptap extensions (which Nexul uses: Yjs collaboration, mention, lowlight code
  blocks, markdown) need the "advanced setup": a separate Vite single-file web
  editor bundle, imported as an HTML string. Because that string is part of the
  JS bundle, it is OTA-updatable, unlike DOM components.
  Sources: https://github.com/10play/10tap-editor, docs
  `setup/advancedSetup.md`, `mainConcepts.md` in the same repo
- `react-native-enriched-html` (Software Mansion, renamed from
  `react-native-enriched`; 1.1.1, 2026-08-14) is a fully native rich-text input
  and display with HTML as the format, New Architecture only. It has no
  Tiptap/Yjs model, so it cannot join a collaborative doc session.
  Source: https://github.com/software-mansion/react-native-enriched-html
- `react-native-enriched-markdown` (1.0.2, 2026-08-20) renders CommonMark + GFM
  (tables, task lists) natively with streaming support, which suits chat.
  Source: https://github.com/software-mansion-labs/react-native-enriched-markdown

**Recommendation.** Chat and read-only docs use `react-native-enriched-markdown`.
Doc editing: phase one embeds the instance's own web editor in
`react-native-webview`. It is a mobile-friendly route on the instance with
the bearer token injected, and it matches the server version automatically,
which removes version skew from the hardest surface. Phase two moves to TenTap
advanced setup built from the same Tiptap extension list as `web/` if the
WebView route feels too much like a website. Avoid DOM components for anything
that must change through OTA.

**Open questions for the owner**

- Is editing docs on a phone a launch requirement, or is reading enough for now?
- Is a web-rendered editor inside the app acceptable UX for v1?

## 8. Push notifications from self-hosted instances

**Findings**

- Expo Push Service: `POST https://exp.host/--/api/v2/push/send` with an
  `ExpoPushToken`. No auth by default. Optional "enhanced security" requires an
  access token, and without it leaked tokens let "a malicious user ...
  impersonate your server". Limits: 600 notifications/s per project, 4096-byte
  payload, receipts checked after about 15 min. It is free: "There is no cost
  associated with sending notifications through Expo push notification
  service." Content "may be seen by Expo staff" while debugging.
  Sources: https://docs.expo.dev/push-notifications/sending-notifications/,
  https://docs.expo.dev/push-notifications/faq/
- APNs/FCM credentials belong to the app's publisher and are uploaded once to
  the Expo project. `getExpoPushTokenAsync` needs the `projectId`. Remote push
  does not work in Expo Go on Android since SDK 53 (a development build is
  required). Source: https://docs.expo.dev/versions/latest/sdk/notifications/
- Precedent for store-distributed apps talking to self-hosted servers:
  Mattermost relays through a hosted push proxy. Hosting your own proxy means
  building your own apps with your own certificates, and an "ID-only" mode sends
  only a message id, with content fetched from the server.
  Source: https://docs.mattermost.com/deployment-guide/mobile/host-your-own-push-proxy-service.
  Rocket.Chat made workspace registration mandatory to use its push gateway.
  Source: https://forums.rocket.chat/t/enforcing-registration-requirement-to-utilize-push-gateway/7545

**Recommendation.** No relay at first. The app registers its `ExpoPushToken`
with the instance it signs into, and the instance posts directly to Expo's
API. Payloads are id-only: instance id, notification id, and a generic title
like "New activity in <project>". The app fetches the content from the
instance on open, so nothing sensitive passes through Expo, Apple or Google. If
abuse or spam appears, switch on Expo's access-token security and put a small
relay on project infrastructure that holds the token and requires instances
to register, as the precedents do. Self-hosters who build their own app can
point instances at their own Expo project.

**Open questions for the owner**

- Is a free, unauthenticated send path acceptable for launch, or should the
  registered-instance relay exist from day one?
- Which events deserve a push (mentions, deploy failed, review requested)?
  That is a server-side catalog decision.

## 9. Sharing code between `web/` and `native/`

**Findings**

- Bun workspaces need a root `package.json` with a `workspaces` field and the
  `workspace:` protocol. Source: https://bun.com/docs/pm/workspaces. Expo
  configures Metro for monorepos automatically since SDK 52 and supports bun
  workspaces. It warns that duplicate React or React Native versions in one app
  "will cause runtime errors". Source: https://docs.expo.dev/guides/monorepos/
- Without workspaces, Metro can still compile files outside the project:
  "all files must be visible to Metro through the combination of `watchFolders`
  and `projectRoot`". A shared file's imports then resolve from the node_modules
  nearest to that file, i.e. `web/node_modules`.
  Source: https://metrobundler.dev/docs/configuration/
- In this repo, 53 of the 56 files in `web/src/models` import only `zod` and
  other models. Three (`DNS`, `Pairing`, `Voice`) import React, UI or stores.
  `web/src/api/client.tsx` imports axios and the session and setup stores, so
  query hooks are coupled to web state. `sdk/` (`@nexul/sdk`) already exports
  `api-client`, `events.generated` and `protocol` with no dependencies.

**Recommendation.** Share only React-free code, and do it without a root
`package.json`. In `native/metro.config.js`, add `../web/src/models` and
`../sdk/src` to `watchFolders`. Add matching `paths` in `native/tsconfig.json`,
and make `zod` resolve from `native/node_modules` through
`resolver.extraNodeModules` so only one copy is bundled. Add a lint rule in
`native/` that fails if a shared file imports React. Write query hooks natively
for now, and extract a React-agnostic `queryOptions` layer later if duplication
hurts. This follows the "no barrels, import by full path" rule because files
are imported directly.

**Open questions for the owner**

- Is a root bun workspace (`web`, `sdk`, `native`, `desktop`) on the table?
  It is the supported path. The cost is one root `package.json` and moving
  lockfiles.
- Should the three React-coupled models be split so all of `models/` is
  shareable?

## 10. Testing and CI

**Findings**

- Unit: `jest-expo` preset (57.0.5; "mocks the native part of the Expo SDK"),
  with bun-specific `transformIgnorePatterns`, plus React Native Testing
  Library (`@testing-library/react-native` 14.0.1), which "replaces the
  deprecated `react-test-renderer`". Tests go outside `app/` because Expo Router
  treats that folder as routes. Source: https://docs.expo.dev/develop/unit-testing/
- E2E: Maestro CLI 2.10.0 (2026-08-31, Apache-2.0) runs YAML flows on
  simulator or emulator builds (`simulator: true` / `buildType: apk`) locally or
  as a `maestro` job in EAS Workflows.
  Sources: https://docs.expo.dev/eas/workflows/examples/e2e-tests/, https://github.com/mobile-dev-inc/Maestro
- Builds: `eas build --local` works on macOS/Linux and still needs `eas login`
  or `EXPO_TOKEN`, with no caching and no secret env vars. `npx expo run:*` is
  the dev-build path. Source: https://docs.expo.dev/build-reference/local-builds/

**Recommendation.** Match the repo's gates: `bun run lint` (eslint via
`expo lint`), `bun run typecheck`, and `bun run test` (jest-expo + RNTL, 80%
floor, error paths first) as a new `native` job behind `dorny/paths-filter`.
Maestro flows for sign-in, instance switch and a push deep link, run locally
on the Android emulator (Linux host) before merge. Jest instead of Vitest is
the one deviation from `web/`, because the Expo preset is Jest-only.

**Open questions for the owner**

- `practices/testing.md` names Vitest. Record Jest for `native/` as an ADR or a
  practices note?
- iOS E2E needs a Mac. Is there one available, or is iOS verified by hand on
  TestFlight?

## 11. Distribution and cost

**Findings**

- EAS Free: up to 15 Android + 15 iOS builds a month, concurrency 1, 1,000
  update MAU, 100 GiB bandwidth. Starter $19/month gives $45 build credit and
  3,000 MAU. No open-source discount is listed. Source: https://expo.dev/pricing
- Apple Developer Program: US$99/year. Source: https://developer.apple.com/programs/whats-included/
- Google Play: one-time US$25 registration. New personal accounts (created after
  2023-11-13) must run a closed test with at least 12 testers opted in for 14
  consecutive days before production access. Organisation accounts are not
  subject to it. Sources: https://support.google.com/googleplay/android-developer/answer/6112435,
  https://support.google.com/googleplay/android-developer/answer/14151465
- App Review needs "an active demo account or fully-featured demo mode" and
  live backend services during review. For a self-hosted product, that means a
  reviewer-reachable demo instance.
  Source: https://developer.apple.com/app-store/review/guidelines/

**Recommendation.** Register both store accounts as the organisation (avoids
the 12-tester rule). Use EAS Build
Free + EAS Submit to TestFlight and the Play internal track, with local
builds as the fallback. Keep a permanent demo instance with a seeded account
for review. Running cost at launch is $99/year + $25 once, and $0 for Expo
while under 1,000 update MAU.

**Open questions for the owner**

- Which legal entity publishes the app?
- Who hosts the review demo instance, and on which domain?

---

## Top risks

1. **Version skew between one app build and many server versions.** This is
   the structural risk of the whole product. Mitigations: additive API, a
   version/capability handshake on connect, per-server-line update channels via
   the header override, and a WebView for the doc editor so the most
   version-sensitive surface comes from the instance itself.
2. **App Review for a self-hosted client.** Review needs a working demo
   backend. Guideline 4.7 makes the publisher responsible for all downloaded JS,
   which is also why per-instance bundles are rejected above.
3. **Push trust model.** The free, direct path through Expo means anyone with a
   leaked push token can notify that device. Id-only payloads limit the damage;
   a registered-instance relay is the fix if it becomes a problem.
