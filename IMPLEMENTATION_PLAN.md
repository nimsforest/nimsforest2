# 🌲 NimsForest Implementation Plan

## Overview

This document contains the ordered implementation plan for NimsForest MVP. Tasks are organized by dependency, not arbitrary timelines.

**Goal:** Automate the route to $1M ARR with 10 FTEs.

**Read first:** [VISION.md](./VISION.md)

---

## Prerequisites

These already exist and are working:

- [x] Core infrastructure (Wind, River, Humus, Soil, Decomposer)
- [x] Base Tree interface and implementation
- [x] Base Nim interface and implementation
- [x] PaymentTree (Stripe webhooks)
- [x] AfterSalesNim (payment followup)
- [x] Leaf types (PaymentCompleted, PaymentFailed, FollowupRequired, EmailSend)
- [x] E2E test infrastructure
- [x] Embedded NATS server

---

## Phase 1: Tree House Foundation

**Goal:** Enable deterministic rule processing.

**Dependency:** None (builds on existing core)

### Tasks

- [ ] **1.1** Create TreeHouse interface
  - File: `internal/core/treehouse.go`
  - Interface: `Name()`, `InputSubjects()`, `Process(leaf) ([]Leaf, error)`, `Start()`, `Stop()`
  - Process must be pure: no side effects, no external calls, no randomness

- [ ] **1.2** Create BaseTreeHouse implementation
  - File: `internal/core/treehouse.go`
  - Catches leaves from Wind matching InputSubjects
  - Calls Process() for each leaf
  - Drops output leaves onto Wind
  - Handles errors gracefully

- [ ] **1.3** Create TreeHouse test helpers
  - File: `internal/core/treehouse_test.go`
  - Helper to assert determinism: same input produces same output
  - Helper to test multiple inputs/outputs

- [ ] **1.4** Register TreeHouse startup in main.go
  - File: `cmd/forest/main.go`
  - Pattern for starting tree houses alongside trees and nims

### Deliverable

Tree Houses can be created, tested, and plugged into the system.

---

## Phase 2: LLM Integration

**Goal:** Enable Nims to use LLM reasoning.

**Dependency:** None (parallel with Phase 1)

### Tasks

- [ ] **2.1** Create LLM client interface
  - File: `internal/llm/client.go`
  - Interface: `Complete(ctx, prompt) (string, error)`
  - Interface: `CompleteStructured(ctx, prompt, schema) (any, error)` for typed responses

- [ ] **2.2** Create OpenAI implementation
  - File: `internal/llm/openai.go`
  - Uses OpenAI API (GPT-4o or similar)
  - Handles rate limits, retries, errors
  - Configurable via environment variables

- [ ] **2.3** Create mock LLM for testing
  - File: `internal/llm/mock.go`
  - Returns predefined responses for testing
  - Allows deterministic nim tests

- [ ] **2.4** Create prompt templates directory
  - Directory: `internal/llm/prompts/`
  - Text files with reusable prompt templates
  - Support for variable substitution

- [ ] **2.5** Create response parser utilities
  - File: `internal/llm/parser.go`
  - Extract structured data from LLM responses
  - Handle JSON, key-value, and freeform formats

### Deliverable

Nims can call LLMs for reasoning. Tests use mock LLM.

---

## Phase 3: Expanded Leaf Types

**Goal:** Define contracts for all MVP events.

**Dependency:** None (parallel with Phases 1-2)

### Tasks

- [ ] **3.1** Add billing/payment leaves
  - File: `internal/leaves/types.go`
  - `PaymentRetryScheduled`, `PaymentRecovered`, `PaymentAbandoned`
  - `DunningEmailSent`, `DunningEscalated`

- [ ] **3.2** Add support leaves
  - File: `internal/leaves/types.go`
  - `TicketCreated`, `TicketValidated`, `TicketRouted`
  - `TicketTriaged`, `TicketEscalated`, `ResponseDrafted`

- [ ] **3.3** Add CRM/sales leaves
  - File: `internal/leaves/types.go`
  - `ContactCreated`, `ContactUpdated`, `ContactScored`
  - `LeadQualified`, `DealCreated`, `DealAnalyzed`

- [ ] **3.4** Add onboarding leaves
  - File: `internal/leaves/types.go`
  - `CustomerOnboarded`, `MilestoneReached`, `MilestoneStuck`
  - `HealthScoreUpdated`, `ChurnRiskDetected`

- [ ] **3.5** Add decision leaves
  - File: `internal/leaves/types.go`
  - `ApprovalRequired`, `ApprovalDecided`
  - `AlertTriggered`, `ThresholdBreached`

### Deliverable

All event contracts defined. Components can communicate.

---

## Phase 4: Billing Automation (Don't Lose Money)

**Goal:** Auto-recover failed payments, alert on billing issues.

**Dependency:** Phase 1 (TreeHouse foundation)

### Tasks

- [ ] **4.1** Create DunningHouse
  - File: `internal/treehouses/dunning.go`
  - Input: `payment.failed`
  - Rules:
    - First failure → schedule retry in 24h
    - Second failure → send dunning email
    - Third failure → escalate to human
  - Output: `payment.retry.scheduled`, `dunning.email.send`, `dunning.escalated`

- [ ] **4.2** Create DunningHouse tests
  - File: `internal/treehouses/dunning_test.go`
  - Test each retry stage
  - Test determinism (same input = same output)

- [ ] **4.3** Create ThresholdHouse
  - File: `internal/treehouses/threshold.go`
  - Input: various metrics/events
  - Rules:
    - Payment failure rate > 10% → alert
    - Customer inactive > 30 days → churn risk
    - Support ticket age > SLA → escalate
  - Output: `alert.triggered`, `threshold.breached`

- [ ] **4.4** Create ThresholdHouse tests
  - File: `internal/treehouses/threshold_test.go`
  - Test each threshold rule
  - Test edge cases (exactly at threshold)

- [ ] **4.5** Wire billing automation into main.go
  - File: `cmd/forest/main.go`
  - Start DunningHouse and ThresholdHouse

### Deliverable

Failed payments auto-retry. Billing issues surface automatically.

---

## Phase 5: Support Tree + Tree Houses

**Goal:** Ingest support tickets, route them automatically.

**Dependency:** Phase 1 (TreeHouse foundation), Phase 3 (leaf types)

### Tasks

- [ ] **5.1** Create SupportTree
  - File: `internal/trees/support.go`
  - Watches: `river.support.zendesk.>`, `river.support.intercom.>`
  - Parses webhook payloads from support platforms
  - Emits: `ticket.created`

- [ ] **5.2** Create SupportTree tests
  - File: `internal/trees/support_test.go`
  - Test Zendesk webhook parsing
  - Test Intercom webhook parsing
  - Test malformed data handling

- [ ] **5.3** Create ValidationHouse
  - File: `internal/treehouses/validation.go`
  - Input: `ticket.created`
  - Rules:
    - Required fields present (customer_id, subject, body)
    - Valid format (email, etc.)
    - Deduplication check
  - Output: `ticket.validated` or `ticket.invalid`

- [ ] **5.4** Create RoutingHouse
  - File: `internal/treehouses/routing.go`
  - Input: `ticket.validated`
  - Rules:
    - Keywords → category (billing, technical, sales)
    - Customer tier → support tier
    - Channel → priority adjustment
  - Output: `ticket.routed` with queue, priority, category

- [ ] **5.5** Create EscalationHouse
  - File: `internal/treehouses/escalation.go`
  - Input: `ticket.routed`, time-based triggers
  - Rules:
    - Ticket age > SLA → escalate
    - Priority P1 + no response > 1h → alert
  - Output: `ticket.escalated`, `alert.triggered`

- [ ] **5.6** Tests for all support tree houses
  - Files: `internal/treehouses/*_test.go`

- [ ] **5.7** Wire support components into main.go

### Deliverable

Support tickets auto-ingest and route to correct queues.

---

## Phase 6: Support Nims (LLM-Powered)

**Goal:** LLM triages tickets and drafts responses.

**Dependency:** Phase 2 (LLM integration), Phase 5 (support infrastructure)

### Tasks

- [ ] **6.1** Create TriageNim
  - File: `internal/nims/triage.go`
  - Catches: `ticket.routed`
  - LLM analyzes:
    - Sentiment (positive, neutral, negative, angry)
    - Urgency (low, medium, high, critical)
    - Intent (question, complaint, request, feedback)
    - Suggested category refinement
  - Emits: `ticket.triaged` with LLM analysis

- [ ] **6.2** Create triage prompt template
  - File: `internal/llm/prompts/triage.txt`
  - Structured prompt for consistent analysis

- [ ] **6.3** Create TriageNim tests
  - File: `internal/nims/triage_test.go`
  - Use mock LLM
  - Test various ticket types

- [ ] **6.4** Create ResponseNim
  - File: `internal/nims/response.go`
  - Catches: `ticket.triaged`
  - LLM drafts response based on:
    - Ticket content
    - Customer history (from Soil)
    - Sentiment-appropriate tone
  - Emits: `response.drafted` (for human review)

- [ ] **6.5** Create response prompt template
  - File: `internal/llm/prompts/response.txt`

- [ ] **6.6** Create ResponseNim tests
  - File: `internal/nims/response_test.go`

- [ ] **6.7** Wire support nims into main.go

### Deliverable

Support tickets auto-triaged by LLM. Response drafts generated for human review.

---

## Phase 7: CRM/Sales Automation

**Goal:** Auto-qualify leads, score contacts.

**Dependency:** Phase 1 (TreeHouse foundation), Phase 3 (leaf types)

### Tasks

- [ ] **7.1** Create CRMTree
  - File: `internal/trees/crm.go`
  - Watches: `river.crm.hubspot.>`, `river.crm.salesforce.>`
  - Parses CRM webhook payloads
  - Emits: `contact.created`, `contact.updated`, `deal.created`

- [ ] **7.2** Create CRMTree tests
  - File: `internal/trees/crm_test.go`

- [ ] **7.3** Create ScoringHouse
  - File: `internal/treehouses/scoring.go`
  - Input: `contact.created`, `contact.updated`
  - Rules (example lead score formula):
    - +20 if company size > 50
    - +30 if title contains "Director", "VP", "CEO"
    - +10 per page view
    - +50 if pricing page viewed
    - +25 if demo requested
  - Output: `contact.scored` with lead_score

- [ ] **7.4** Create QualificationHouse
  - File: `internal/treehouses/qualification.go`
  - Input: `contact.scored`
  - Rules:
    - Score >= 50 → MQL
    - Score >= 80 + demo requested → SQL
  - Output: `lead.qualified` with qualification level

- [ ] **7.5** Tests for CRM tree houses
  - Files: `internal/treehouses/*_test.go`

- [ ] **7.6** Create DealNim (optional for MVP)
  - File: `internal/nims/deal.go`
  - LLM analyzes deal and suggests next action
  - Lower priority than support nims

- [ ] **7.7** Wire CRM components into main.go

### Deliverable

Leads auto-scored and qualified. Sales talks to ready-to-buy prospects.

---

## Phase 8: Onboarding Automation

**Goal:** Customers self-onboard successfully.

**Dependency:** Phase 1 (TreeHouse foundation)

### Tasks

- [ ] **8.1** Create OnboardingHouse
  - File: `internal/treehouses/onboarding.go`
  - Input: `payment.completed` (new customer)
  - Rules:
    - Trigger welcome email sequence
    - Set onboarding milestones
    - Track progress
  - Output: `customer.onboarded`, `onboarding.step.triggered`

- [ ] **8.2** Create MilestoneHouse
  - File: `internal/treehouses/milestone.go`
  - Input: usage events, `onboarding.step.completed`
  - Rules:
    - Track activation milestones (first login, first action, etc.)
    - If stuck > X days → nudge
    - If complete → mark activated
  - Output: `milestone.reached`, `milestone.stuck`, `customer.activated`

- [ ] **8.3** Create HealthScoreHouse
  - File: `internal/treehouses/healthscore.go`
  - Input: various customer activity events
  - Rules (example health score):
    - +20 if logged in last 7 days
    - +30 if core feature used
    - -20 if support ticket open
    - -40 if payment failed
  - Output: `healthscore.updated`, `churn.risk.detected` if score drops

- [ ] **8.4** Tests for onboarding tree houses

- [ ] **8.5** Wire onboarding components into main.go

### Deliverable

Customers guided through onboarding. At-risk customers identified automatically.

---

## Phase 9: Human Approval Workflow

**Goal:** High-value decisions routed to humans.

**Dependency:** Phase 3 (leaf types)

### Tasks

- [ ] **9.1** Create ApprovalNim
  - File: `internal/nims/approval.go`
  - Catches: `approval.required`
  - Stores pending approval in Soil
  - Emits: `approval.pending`
  - Waits for external trigger (API/UI) → `approval.decided`

- [ ] **9.2** Create approval callback mechanism
  - API endpoint or NATS subject for humans to submit decisions
  - Updates Soil, emits `approval.decided`

- [ ] **9.3** Create ApprovalNim tests
  - File: `internal/nims/approval_test.go`

- [ ] **9.4** Wire approval workflow into main.go

### Deliverable

Exceptions queue for human decision. Humans can approve/reject via API.

---

## Phase 10: Integration & Testing

**Goal:** Everything works together.

**Dependency:** All previous phases

### Tasks

- [ ] **10.1** E2E test: Payment flow
  - Stripe webhook → PaymentTree → DunningHouse (if failed) → Soil
  - Test success and failure paths

- [ ] **10.2** E2E test: Support flow
  - Zendesk webhook → SupportTree → ValidationHouse → RoutingHouse → TriageNim → ResponseNim
  - Test full ticket lifecycle

- [ ] **10.3** E2E test: CRM flow
  - HubSpot webhook → CRMTree → ScoringHouse → QualificationHouse
  - Test lead scoring and qualification

- [ ] **10.4** E2E test: Onboarding flow
  - New customer → OnboardingHouse → MilestoneHouse → HealthScoreHouse
  - Test activation tracking

- [ ] **10.5** Update demo mode
  - File: `cmd/forest/main.go`
  - Send realistic sample data through all flows
  - Show complete system in action

- [ ] **10.6** Update README with new components
  - Document all new trees, tree houses, and nims

### Deliverable

Production-ready system with comprehensive tests.

---

## File Checklist

### Core

- [ ] `internal/core/treehouse.go`
- [ ] `internal/core/treehouse_test.go`

### LLM

- [ ] `internal/llm/client.go`
- [ ] `internal/llm/openai.go`
- [ ] `internal/llm/mock.go`
- [ ] `internal/llm/parser.go`
- [ ] `internal/llm/prompts/triage.txt`
- [ ] `internal/llm/prompts/response.txt`

### Leaves

- [ ] `internal/leaves/types.go` (expand)

### Trees

- [x] `internal/trees/payment.go` ✅
- [ ] `internal/trees/support.go`
- [ ] `internal/trees/support_test.go`
- [ ] `internal/trees/crm.go`
- [ ] `internal/trees/crm_test.go`

### Tree Houses

- [ ] `internal/treehouses/dunning.go`
- [ ] `internal/treehouses/dunning_test.go`
- [ ] `internal/treehouses/threshold.go`
- [ ] `internal/treehouses/threshold_test.go`
- [ ] `internal/treehouses/validation.go`
- [ ] `internal/treehouses/validation_test.go`
- [ ] `internal/treehouses/routing.go`
- [ ] `internal/treehouses/routing_test.go`
- [ ] `internal/treehouses/escalation.go`
- [ ] `internal/treehouses/escalation_test.go`
- [ ] `internal/treehouses/scoring.go`
- [ ] `internal/treehouses/scoring_test.go`
- [ ] `internal/treehouses/qualification.go`
- [ ] `internal/treehouses/qualification_test.go`
- [ ] `internal/treehouses/onboarding.go`
- [ ] `internal/treehouses/onboarding_test.go`
- [ ] `internal/treehouses/milestone.go`
- [ ] `internal/treehouses/milestone_test.go`
- [ ] `internal/treehouses/healthscore.go`
- [ ] `internal/treehouses/healthscore_test.go`

### Nims

- [x] `internal/nims/aftersales.go` ✅
- [ ] `internal/nims/triage.go`
- [ ] `internal/nims/triage_test.go`
- [ ] `internal/nims/response.go`
- [ ] `internal/nims/response_test.go`
- [ ] `internal/nims/approval.go`
- [ ] `internal/nims/approval_test.go`
- [ ] `internal/nims/deal.go` (optional)

### Tests

- [x] `test/e2e/forest_test.go` ✅
- [ ] `test/e2e/support_test.go`
- [ ] `test/e2e/crm_test.go`
- [ ] `test/e2e/onboarding_test.go`

---

## Getting Started

Begin with **Phase 1, Task 1.1**: Create the TreeHouse interface.

This unblocks all tree house development and establishes the deterministic processing pattern that is core to the architecture.
