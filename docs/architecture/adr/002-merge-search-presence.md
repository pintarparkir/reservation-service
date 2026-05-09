# ADR-003: Merge `search` and `presence` services

**Status:** Accepted  
**Date:** 2026-04-27

## Context
Soal suggests services: gateway, search, reservation, billing, payment, presence, notification — and explicitly allows merging with justification.

## Decision
Merge `search` + `presence` into `presence-search` service.

## Why
1. **Same data shape** — both query inventory state (spots, occupancy counters). One Redis cache, one DB read pool.
2. **Both read-mostly** — no state machine overlap with other services.
3. **No write coupling** — neither owns reservation lifecycle; they only project the read view.
4. **Cost** — saves ~1 Cloud Run service min instance (~$8/mo) and 1 DB connection pool.
5. **Latency** — eliminates 1 extra gRPC hop for the common "show me spots near me" call.

## Trade-off
- **−** Slightly larger blast radius if this service has a bug.
- **−** Two engineers can't independently iterate on search/presence (low risk at MVP team size).

## Trigger to split
- Either responsibility grows beyond ~2k LOC of distinct logic.
- Search adds ML ranking/personalization (different deployment cadence).
