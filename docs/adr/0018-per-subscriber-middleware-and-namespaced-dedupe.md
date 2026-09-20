# The bus builds its middleware per subscriber, and dedupe keys are namespaced per consumer

The in-process bus built its middleware stack once in `New` and shared the
instances across every subscriber. With dedupe enabled the first subscriber to
run recorded the bare event ID and every other subscriber of the same topic
was silently skipped, and one global throttle semaphore serialized the whole
bus. The chain is now built per subscription: the semaphore is per
subscription (which is what `ThrottleLimit` always claimed), and dedupe keys
are namespaced `"<consumer>:<event_id>"` through a store wrapper, with
`SubscribeWithConsumer` supplying an identity that is explicit in the
composition root and therefore stable across restarts. Namespacing the key
beat migrating `processed_events` to a composite key: same table shape, no
migration, and a redelivery to the same subscriber still dedupes.

Separately, `buildEvent` stamped a fresh ID on every publish and the outbox
relay discarded the row's own ID, so a crash-redelivered row could never be
deduped. The relay and dead-letter replay now publish under the source row's
ID via a narrow `PublishWithID`; the `EventBus` interface itself is unchanged.

Decided: 2026-08-13
