#!/usr/bin/env bash
# Two-phase e2e runner: the onboarding spec flips wizard flags on the seeded
# user, so it runs alone AFTER the main suite (vitest 3.2.x cannot serialize
# test files — see vitest.e2e.config.ts). Fails if either phase fails.
set -o pipefail

# Warm the deploy fixture so the runner's `docker pull` never blocks on a
# cold registry fetch mid-test.
docker pull nginx:alpine >/dev/null 2>&1 || true

bunx vitest run --config vitest.e2e.config.ts --exclude e2e/specs/onboarding.spec.ts
main_status=$?

bunx vitest run --config vitest.e2e.config.ts e2e/specs/onboarding.spec.ts
onboarding_status=$?

# Deploy tests leave real containers on the host docker daemon (each named
# after its e2e-* service); reap them so reruns never collide.
docker ps -aq --filter 'name=^e2e-' | xargs -r docker rm -f >/dev/null 2>&1 || true

if [[ $main_status -ne 0 || $onboarding_status -ne 0 ]]; then
  exit 1
fi
