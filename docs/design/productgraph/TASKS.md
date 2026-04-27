# ProductGraph Integration Tasks

**Last Updated:** 2026-04-27

## Overview

Task breakdown for integrating coreforge observability with ProductGraph.

## Phase 1: Correlation Middleware

### P1.1: Design Documents

- [x] Create PRD.md with product requirements
- [x] Create TRD.md with technical specification
- [x] Create PLAN.md with implementation phases
- [x] Create TASKS.md (this file)

### P1.2: Correlation Package

- [ ] Create observability/correlation/correlation.go
- [ ] Implement Middleware function
- [ ] Implement SessionIDFromContext helper
- [ ] Implement RequestIDFromContext helper
- [ ] Add header validation (UUID format)
- [ ] Create observability/correlation/correlation_test.go
- [ ] Test middleware extracts headers
- [ ] Test context helpers

## Phase 2: ProductGraph Client

### P2.1: Core Client

- [ ] Create productgraph/config.go
- [ ] Define Config struct with defaults
- [ ] Implement ConfigFromEnv function
- [ ] Create productgraph/event.go
- [ ] Define Event struct with OTel fields
- [ ] Define event type constants
- [ ] Create productgraph/client.go
- [ ] Implement New constructor
- [ ] Implement Track method
- [ ] Implement async batching
- [ ] Implement flush goroutine
- [ ] Implement Close with graceful shutdown

### P2.2: Convenience Methods

- [ ] Implement TrackAPICall method
- [ ] Implement TrackError method
- [ ] Implement TrackJourneyStep method
- [ ] Add context session ID extraction

### P2.3: Client Tests

- [ ] Create productgraph/client_test.go
- [ ] Test event batching
- [ ] Test flush on batch size
- [ ] Test flush on interval
- [ ] Test flush on close
- [ ] Test HTTP request format
- [ ] Test API key header
- [ ] Test session ID from context

## Phase 3: Observability Integration

### P3.1: Provider Integration

- [ ] Add productgraph field to Provider struct
- [ ] Implement WithProductGraph option
- [ ] Add ProductGraph to provider initialization
- [ ] Add Close to provider shutdown

### P3.2: Request Tracking Middleware

- [ ] Implement RequestTracker middleware
- [ ] Capture response status code
- [ ] Capture request duration
- [ ] Track api.response events
- [ ] Add to middleware chain docs

### P3.3: Integration Tests

- [ ] Test full middleware chain
- [ ] Test correlation + tracking
- [ ] Test with mock ProductGraph server
- [ ] Test graceful shutdown

## Phase 4: Documentation

### P4.1: Package Documentation

- [ ] Create productgraph/README.md
- [ ] Document configuration options
- [ ] Document event schema
- [ ] Add usage examples

### P4.2: Integration Guide

- [ ] Document middleware chain setup
- [ ] Document environment variables
- [ ] Document correlation with frontend
- [ ] Add troubleshooting section

### P4.3: Example Code

- [ ] Create productgraph/example_test.go
- [ ] Example: Basic tracking
- [ ] Example: Journey tracking
- [ ] Example: Error tracking
- [ ] Example: Full middleware chain

## Backlog

### Future Enhancements

- [ ] Retry with exponential backoff
- [ ] Gzip compression for batches
- [ ] Circuit breaker for failures
- [ ] Metrics for tracking (events sent, failures)
- [ ] Health check endpoint contribution
- [ ] OpenTelemetry span linking
- [ ] Sampling support
- [ ] PII redaction filters

## Completed

- [x] Design documents created (PRD, TRD, PLAN, TASKS)

## Notes

### Priority Legend

- P0: Critical path, blocks release
- P1: Important, should have for release
- P2: Nice to have
- P3: Future consideration

### Current Focus

Phase 1: Design documents and correlation middleware.

### Blockers

None currently identified.

### Dependencies on Other Work

- ProductGraph v0.2.0 must be released (DONE)
- @coreforge/telemetry ProductGraphAdapter provides correlation headers

## Test Coverage Goals

| Package | Target |
|---------|--------|
| correlation | 90% |
| productgraph | 85% |
| observability (new code) | 80% |

## Related Documents

- [PRD.md](PRD.md) - Product requirements
- [TRD.md](TRD.md) - Technical requirements
- [PLAN.md](PLAN.md) - Implementation plan
- [Observability TRD](../FEAT_OBSERVABILITY_TRD.md)
