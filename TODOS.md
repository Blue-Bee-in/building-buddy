# TODOS.md
Deferred work from plan reviews. Updated by /autoplan on 2026-05-03.

## Deferred (Post-Phase 3)

### WhatsApp Bot Layer
Source: /office-hours design doc + /plan-ceo-review
Defer until: WhatsApp integration demand is validated by secretary interviews. Platform risk (Meta BSP dependency, template approval, iOS UPI incompatibility) must be evaluated first. Outside voice confirmed iOS UPI deep-links do not work from WhatsApp.

### RWA Federation Tier
Source: /plan-ceo-review cherry-pick (DEFERRED)
Multi-society view for property management firms. Phase 4+ only. Requires paying customer base first. The `society_id` on every table supports this architecturally — it's a read-across-societies query layer.

## Open Questions (blocking future phases)

### Distribution / GTM channel
Source: outside voice review
Blocker for: Phase 2 sales motion
Question: How does society #1 get onboarded? Society #2? Indian secretaries (50-65, elected volunteers) are not reachable via SEO or cold email. Incumbents (MyGate, NoBrokerHood) used on-ground AGM sales reps. Name the acquisition channel before Phase 2 build begins.

### Pricing approval dynamics
Source: outside voice review
Blocker for: Phase 2 pricing decisions
Question: Recurring software spend >Rs.1,000/month typically requires AGM approval in Indian housing societies. Sales cycle could be 3-12 months per society if committee approval is needed. Decide: secretary pays out-of-pocket (lower price, faster close) or committee-approved (higher price, longer cycle). This affects pricing model, sales motion, and Razorpay KYC entity.

### Audit log
Source: /office-hours review (inherited)
Blocker for: Phase 1 acceptance criteria ("all events persisted in secretary-only audit log")
Events to audit: maintenance request state transitions, broadcast sends, payment records.
Schema needs: which events, what log entry fields, where in mobile/web split the log lives.

## iOS Payment Path (Phase 2)
Source: outside voice review
UPI deep-links embedded in WhatsApp messages do NOT work on iOS (GPay/PhonePe don't open from WA on iOS). Design a non-WhatsApp payment collection path for iOS residents before Phase 2 billing ships. Options: in-app payment button (Razorpay SDK), SMS deep-link, QR code in PDF.

## Audio Chunking Decision (before Phase 2 ends)
Source: /autoplan CEO + Eng review
Blocker for: Phase 3 AI minutes sprint start
Decision needed: client-side chunking (React Native native audio module, not in dep tree) vs server-side chunking (FFmpeg subprocess, significant binary dep). Neither path has an owner. Must be decided before Phase 2 ends so Phase 3 sprint can start without a blocked dependency.

## `meeting_minutes` Schema Design (before manual minutes coding)
Source: /autoplan Eng review
Blocker for: Phase 3 meeting minutes
Manual minutes and AI minutes must share one DB schema and edit UI. If manual minutes ships with an ad-hoc schema, the AI minutes JSON output must conform. Design the `meeting_minutes` table (agenda items, vote tallies, resolution fields) before any Phase 3 minutes coding starts to avoid mid-sprint schema refactor.

## Razorpay Webhook HMAC-SHA256 Verification (Phase 2, first webhook handler)
Source: /autoplan Eng review
Security requirement: Razorpay sends a signature header `X-Razorpay-Signature`. Handler must verify HMAC-SHA256 before processing any event. This is Razorpay's documented requirement and should be the first check in any webhook handler, not an afterthought.

## `monthly_collection_summary(society_id, month)` Query Contract (Phase 2)
Source: /autoplan CEO plan (Phase 2-3 interface contract)
Phase 3 health report River job calls `monthly_collection_summary(society_id, month) → (total_charged, total_collected)`. Phase 2 billing schema must expose this query so Phase 3 has a stable interface. Note it in Phase 2 sprint planning so the health report isn't blocked at Phase 3 start.

## `formatINR` TypeScript Port (Phase 1 mobile)
Source: /autoplan Design review
The Go `internal/pdf/pdf.go` has `formatINR()` for INR lakh format (`₹1,25,000`). Mobile app needs the same function for displaying quoted/approved amounts. Port to TypeScript as a shared utility before the maintenance form is built.
