# Building Buddy — Flow Diagrams

This document covers every major user journey in Building Buddy, including actors, step-by-step flows, edge cases, and links to the FigJam diagrams.

## Actors

| Actor | Surface | Role |
|---|---|---|
| Super Admin | Web | Platform-level administrator across all societies |
| Society Admin | Web | Full control within their society |
| Committee Member | Web | Read-only financials; can post notices and manage complaints |
| Security Guard | Mobile (gate app) | Logs visitor, delivery, and domestic help entry/exit |
| Resident | Mobile | Pays dues, pre-approves visitors, raises complaints |
| Visitor / Delivery Agent | — | External party entering the society |

---

## 1. Society Onboarding

**Actors:** Super Admin, Society Admin  
**Linear:** BEE-6, BEE-7, BEE-8, BEE-9, BEE-20, BEE-21  
**FigJam:** https://www.figma.com/online-whiteboard/create-diagram/a52f0d05-df41-4902-bbf5-302b37bab1ec

### Flow

1. Super Admin creates the society record on the web portal (name, address, city)
2. Society Admin fills in the flat structure — blocks, floors, unit numbers
3. Bank details submitted for Razorpay KYC verification
4. On approval, a Razorpay Linked Account is created so payments flow directly to the society's bank
5. Admin configures charge heads (maintenance, sinking fund, parking, etc.)
6. Flat directory populated with initial resident data
7. Society status set to **Live** — residents can now be invited

### Edge Cases

- **Razorpay KYC rejected** — admin re-submits corrected bank details; society stays in draft state until approved
- **Duplicate society** — system checks for existing society at the same address before creation
- **Admin leaves** — Super Admin can reassign the Society Admin role to another user

---

## 2. Resident Onboarding

**Actors:** Society Admin, Resident  
**Linear:** BEE-10, BEE-20, BEE-21  
**FigJam:** https://www.figma.com/online-whiteboard/create-diagram/92c25dd7-2b64-4804-a316-7ea65d2d9475

### Flow

1. Society Admin adds a resident to a flat (name, phone number, owner vs. tenant)
2. System generates a unique deep link invite
3. Invite sent to resident's phone via WhatsApp/SMS
4. Resident taps link — if app is not installed, redirected to Play Store / App Store
5. App opens on the registration screen with flat details pre-filled from the invite
6. Resident sets their password and completes profile
7. JWT issued containing `user_id`, `society_id`, `flat_id`, and `role=resident`
8. Resident lands on the home screen

### Edge Cases

- **Invite link expired** — admin can re-generate and resend from the web portal
- **Resident already registered** — link shows a login prompt instead of registration
- **Multiple residents per flat** — each gets their own invite and account; all linked to the same flat

---

## 3. Billing & Payment

**Actors:** Go Background Worker, Resident, Society Admin  
**Linear:** BEE-11, BEE-12, BEE-13, BEE-14, BEE-15, BEE-22, BEE-23, BEE-24  
**FigJam:** https://www.figma.com/online-whiteboard/create-diagram/63661476-e3b9-4d9f-b078-3cca34b413fc

### Flow

1. Go Ticker runs daily and checks each society's billing cycle
2. Identifies flats due for billing in the current cycle
3. Fetches all applicable charge heads for each flat (maintenance, sinking fund, etc.)
4. Checks for overdue invoices — appends penalty/interest if configured grace period has elapsed
5. Creates invoice record in Postgres with status `PENDING`
6. PDF generated via `gofpdf` with itemised charge breakdown
7. PDF uploaded to DigitalOcean Spaces; signed URL stored on the invoice record
8. WhatsApp message sent to resident with invoice link and UPI intent deep-link (GPay/PhonePe)
9. Resident taps UPI link and completes payment
10. Razorpay webhook hits `/webhooks/razorpay` on the API
11. API verifies Razorpay signature
12. Invoice status updated to `PAID`; payment record created in ledger
13. Receipt PDF generated and sent to resident via WhatsApp

### Edge Cases

- **Webhook signature invalid** — payment logged as suspicious; admin alerted; invoice stays `PENDING`
- **Partial payment** — invoice status set to `PARTIALLY_PAID`; remainder carried forward
- **Offline payment (cash/cheque)** — admin manually records payment via web portal (BEE-23); audit trail stored
- **Billing cycle change** — charge head changes only apply from the next cycle, never retroactively

---

## 4. Visitor

**Actors:** Resident, Security Guard, Visitor  
**Linear:** BEE-25, BEE-26  
**FigJam:** https://www.figma.com/online-whiteboard/create-diagram/4cfbcc63-e3ef-4e71-95cd-8e3aecb80ad0

### Flow

1. Resident creates a visitor pre-approval in the mobile app (name, phone, date/time, single-use or recurring)
2. System generates a unique QR code and 6-digit OTP
3. QR/OTP sent to visitor's phone via WhatsApp/SMS
4. Visitor arrives at gate and presents QR or OTP to guard
5. Guard scans QR or types OTP in the gate app
6. System validates — checks expiry and whether pass has already been used
7. On success: entry logged with timestamp, flat number, and visitor details
8. Resident receives push notification: "Your visitor [name] has entered"
9. Visitor exits — guard taps "Exit" in the gate app
10. Exit timestamp recorded; visit record closed

### Edge Cases

- **Pass expired** — entry denied; resident notified of the failed attempt so they can re-approve
- **Pass already used** — single-use passes are invalidated immediately after first scan
- **Walk-in visitor (no pre-approval)** — guard triggers a call-to-resident confirmation prompt; resident approves or denies from the app
- **Overstay** — if visitor remains beyond the configured duration, push alert sent to guard and resident

---

## 5. Delivery & Package

**Actors:** Resident, Security Guard, Delivery Agent  
**Linear:** BEE-27  
**FigJam:** https://www.figma.com/online-whiteboard/create-diagram/0d73035f-19aa-47c0-a86d-a568387f5af4

### Flow

1. Resident optionally pre-approves an expected delivery (courier name, time window)
2. Delivery agent arrives at gate
3. Guard logs: courier/platform name, destination flat, photo of package
4. Resident notified immediately via push notification and WhatsApp
5. **If resident is available** — guard allows direct delivery to flat; agent enters on a time-limited pass
6. **If resident is unavailable** — package marked as "held at gate" with photo evidence
7. Resident taps "Acknowledge Pickup" in the app when they collect
8. Guard confirms handover; delivery agent exit logged

### Edge Cases

- **Delivery agent tries to re-enter after pass expiry** — blocked; guard alerted
- **Package uncollected for >24 hours** — admin notified to follow up with resident
- **Unrecognised courier** — guard manually logs and notifies resident before allowing entry

---

## 6. Domestic Help

**Actors:** Resident, Security Guard, Domestic Staff  
**Linear:** BEE-28  
**FigJam:** https://www.figma.com/online-whiteboard/create-diagram/b43c9fcb-f3ee-4d91-9f53-a6fea78ba74b

### Flow

1. Resident registers domestic help in the mobile app (name, phone, photo, type, working days/hours)
2. Optional: upload identity documents (Aadhaar, police verification) for society records
3. Guard app refreshed with the "expected today" list based on working days
4. Help arrives at gate — guard sees them on the expected list and marks entry in one tap
5. Resident notified: "[Name] (maid) has arrived"
6. Help exits at end of shift — guard marks exit
7. Attendance recorded; monthly report available to resident in the app

### Edge Cases

- **Help arrives on an unexpected day** — not on the expected list; guard calls resident for confirmation
- **Society-wide blacklist** — admin can block a specific individual from entering any flat; guard app flags the attempt
- **Resident deactivates help profile** — person removed from expected list immediately; cannot enter until re-registered

---

## 7. Complaint & Helpdesk

**Actors:** Resident, Society Admin, Committee Member, Maintenance Staff  
**Linear:** BEE-29  
**FigJam:** https://www.figma.com/online-whiteboard/create-diagram/5a93c6af-dde4-4aba-93e5-e6bb84778bc1

### Flow

1. Resident raises a complaint in the mobile app (category, description, optional photo)
2. Ticket created with status `Open`; admin notified on the web dashboard
3. Admin assigns ticket to a staff member or vendor with an expected resolution date
4. Status updated to `In Progress`; resident notified
5. **SLA check runs continuously** — if deadline is breached, ticket is auto-escalated to Society Admin
6. Staff member resolves the issue and marks ticket `Resolved`
7. Resident receives push notification and is prompted to confirm resolution
8. **If resident confirms** — ticket closed
9. **If resident disagrees** — resident reopens the ticket; it re-enters the assignment queue

### Edge Cases

- **SLA breach** — auto-escalation to Society Admin with breach timestamp; feeds into admin reporting dashboard
- **Recurring issue** — admin can link multiple tickets to identify repeat problems (e.g. persistent lift fault)
- **Vendor assigned** — vendor receives details via WhatsApp (no app login required)

---

## 8. Emergency SOS

**Actors:** Resident, Security Guard, Society Admin  
**Linear:** BEE-30  
**FigJam:** https://www.figma.com/online-whiteboard/create-diagram/3237525b-db85-4189-b4c9-aa0d5377294d

### Flow

1. Resident taps the SOS button in the mobile app
2. Alert immediately dispatched containing flat number and GPS location
3. High-priority FCM notification sent — bypasses Do Not Disturb on device
4. Security guard notified and dispatched to the flat
5. Society Admin notified simultaneously
6. Admin assesses the situation:
   - **If localised** — guard handles; incident logged
   - **If society-wide threat** — admin sends mass broadcast push notification to all residents (e.g. fire, gas leak)
7. SOS log closed once resolved; response time recorded for admin review

### Edge Cases

- **Accidental SOS** — resident can cancel within 10 seconds; if cancelled, guard/admin are notified it was a false alarm
- **Guard offline** — if guard app is not active, alert is escalated directly to Society Admin and backup contact
- **Mass broadcast** — broadcast notifications are tagged `URGENT` and cannot be muted by residents
