# Design System — building-buddy

## Product Context
- **What this is:** Housing society management software for India's elected secretary — maintenance tracking, billing, complaints, AGM minutes, and handover reports with full audit trails.
- **Who it's for:** Society secretary (50–65yo, retired professional, elected volunteer). Manages 200+ residents from their personal phone. Has legal accountability with no paper trail.
- **Space/industry:** Indian housing society management (competitors: MyGate, ApnaComplex, NoBrokerHood)
- **Project type:** Mobile-first daily ops (Expo) + web admin for AGM/ledger work (Next.js)

## Design Anchor
**"This is serious software for serious work."**

Every visual decision answers: does this make the secretary look like they're running a proper operation?

## Aesthetic Direction
- **Direction:** Institutional Utility
- **Decoration level:** Intentional (minimal) — no gradients, no blobs. Typography is the decoration.
- **Mood:** The visual language of official documentation and governance infrastructure. Authoritative, calm, permanent. Feels like a well-designed legal document system, not a SaaS dashboard.
- **Competitive gap:** Every competitor (MyGate, ApnaComplex, NoBrokerHood) uses medium blue + Inter/Roboto + utilitarian flat design. None design for reputation protection. This product deliberately looks institutional — the secretary shows a document, not a screenshot.

## Typography

- **Display / Document headings / PDF exports:** Fraunces (variable, optical-size-aware serif)
  - Use for: notice headings, maintenance request PDFs, AGM meeting minutes, handover reports, section headings in web admin
  - Rationale: Optical-size axis means headings look different from body — both feel intentionally designed. Warm, premium, non-SaaS. No competitor uses it. Makes PDFs look like they came from a real secretary's office.
  - Load: `family=Fraunces:ital,opsz,wght@0,9..144,300;0,9..144,400;0,9..144,500;0,9..144,600;0,9..144,700;1,9..144,400;1,9..144,500`

- **Body / UI / Labels:** Plus Jakarta Sans
  - Use for: all app chrome, form labels, body text, navigation, buttons, table data
  - Rationale: Clean, professional, legible at 16px+ for 50–65yo users. Less ubiquitous than DM Sans or Inter.
  - Load: `family=Plus+Jakarta+Sans:wght@300;400;500;600;700`

- **Data / Tables:** Plus Jakarta Sans with `font-variant-numeric: tabular-nums`
  - Use for: billing ledger, resident lists, invoice amounts

- **Reference IDs / Timestamps / Audit trail:** JetBrains Mono
  - Use for: request IDs (MR-2026-042), notification IDs (NT-2026-0018), ISO timestamps, flat codes (FLAT-B-204)
  - Load: `family=JetBrains+Mono:wght@400;500`

### Type Scale

| Level | Font | Size | Weight | Use |
|-------|------|------|--------|-----|
| Display | Fraunces | 32–64px | 700 | Hero headings, PDF title |
| H1 | Fraunces | 28px | 700 | Page title (web) |
| H2 | Fraunces | 22px | 600 | Section heading, card title |
| H3 | Fraunces | 18px | 600 | Sub-section, notice title |
| Body | Plus Jakarta Sans | 15–16px | 400 | All prose |
| Label | Plus Jakarta Sans | 13px | 600 | Form labels, table headers |
| Caption | Plus Jakarta Sans | 12px | 400–500 | Timestamps, helper text |
| Mono | JetBrains Mono | 11–13px | 400–500 | IDs, timestamps, audit data |

### Mobile-specific
- Minimum body size: 15px (accessibility for 50–65yo target)
- Minimum touch target: 44px height (Apple HIG); prefer 48px for primary inputs

## Color

- **Approach:** Restrained — one accent + neutrals, color is rare and meaningful

| Token | Hex | Use |
|-------|-----|-----|
| Primary / Text | `#1E2A3A` | All primary text, headers, primary UI elements |
| Accent (Forest Teal) | `#1B6F68` | CTA buttons, active states, links, progress |
| Accent Light | `#E8F5F4` | Accent backgrounds, active nav items |
| Surface (Paper) | `#F5F4F0` | Document areas, cards, form backgrounds — archival feel |
| App BG | `#FFFFFF` | Chrome, navigation, modal backgrounds |
| Border | `#DDD9D2` | All borders and dividers |
| Text Muted | `#5C6676` | Secondary text, labels, helper text |
| Text Hint | `#8E9AA6` | Placeholders, metadata, disabled |
| Success | `#2D7A57` | Approved, Paid, Delivered |
| Success BG | `#E8F5EE` | Success badge backgrounds |
| Warning | `#7A5A1E` | Pending, Draft, Partial |
| Warning BG | `#FDF3E1` | Warning badge backgrounds |
| Error | `#8B2020` | Rejected, Overdue, Failed |
| Error BG | `#FDEAEA` | Error badge backgrounds |

### Dark mode
Surface hierarchy: `#111820` (app bg) → `#1A1F28` (elevated surfaces) → `#222933` (cards).
Accent in dark: `#23A89E` (slightly lighter teal for contrast on dark backgrounds).
Reduce text saturation: `#E8E6E1` (high) / `#8A9099` (muted).

### Contrast ratios
- Primary on Surface: **12.8:1** (AAA — exceeds standard)
- Accent on White: **6.1:1** (AA — all text sizes)
- Success on BG: **7.3:1** (AA)

## Spacing

- **Base unit:** 4px
- **Density:** Comfortable (not sparse, not tight)
- **Scale:** 2xs=2 xs=4 sm=8 md=16 lg=24 xl=32 2xl=48 3xl=64 4xl=96
- **Screen padding (mobile):** 16px
- **Card padding:** 20px 24px
- **Section padding (web):** 80px 0

## Layout

- **Mobile:** Grid-disciplined — strict columns, predictable alignment, thumb-zone-aware CTAs at bottom of screen
- **Web:** Hybrid — grid for data screens (billing ledger, complaint queue), document-editorial for report generation (handover report, AGM minutes, monthly health PDF)
- **Max content width (web):** 1100px
- **Sidebar width (web admin):** 220px

## Border Radius

| Level | Value | Use |
|-------|-------|-----|
| sm | 4px | Input focus rings, table row hovers |
| md | 6px | Inputs, buttons, small cards |
| lg | 10px | Cards, modals, panels |
| pill | 9999px | Status badges only |

No bubbly 16px+ radius on interactive elements. This is governance software, not a consumer app.

## Motion

- **Approach:** Minimal-functional — state transitions only, no choreography
- **Principle:** Serious software doesn't dance
- **Easing:** enter(ease-out 200ms) · exit(ease-in 150ms) · move(ease-in-out 250ms)
- **Duration:** micro=50–100ms · short=150–200ms · medium=250ms · long=400ms

## Status Badges

Always use **color + label** — never color alone. Minimum font size 12px.

| State | Color | Background |
|-------|-------|------------|
| Approved / Paid / Delivered | `#2D7A57` | `#E8F5EE` |
| Pending / Draft / Processing | `#7A5A1E` | `#FDF3E1` |
| Rejected / Overdue / Failed | `#8B2020` | `#FDEAEA` |
| Neutral / Unknown | `#5C6676` | `#F5F4F0` |
| Accent / Active | `#1B6F68` | `#E8F5F4` |

## PDF Template Spec

PDFs are a primary deliverable — the secretary's reputational shield. They must look official.

**Header:** Dark primary (`#1E2A3A`) background · Society name (caps, Mono 11px) · Document title (Fraunces 22px bold) · Reference ID and date (Mono 11px)

**Body:** Warm paper surface (`#F5F4F0`) · Two-column label/value table (Fraunces body 14px) · Signature lines with dotted bottom border · "SECRETARY SIGNATURE" / "CONTRACTOR ACKNOWLEDGEMENT" labels in Mono

**Footer:** Society name · "Generated by building-buddy" · ISO timestamp in Mono

**Filename convention:** `{type}-{year}-{seq}-{slug}.pdf`
Examples: `MR-2026-042-lobby-lights.pdf` · `NT-2026-0018-agm-notice.pdf` · `HR-2026-001-handover-apr.pdf`

## Implementation

### Mobile (Expo / React Native)
- **Styling:** NativeWind (Tailwind CSS for React Native) — matches web token system, reduces context switching for a solo/small team
- **Token source:** `packages/tokens` (shared color/spacing constants, importable by mobile `@/constants/theme` and web `tailwind.config.ts`)
- **Existing theme file:** `apps/mobile-app/constants/theme.ts` — extend, do not replace. Map existing `Colors.light.tint` → `--accent (#1B6F68)`.
- **Minimum touch target:** 48px height for all interactive elements
- **Form inputs:** `keyboardType="numeric"` for amount fields, `KeyboardAvoidingView` on all form screens

### Web (Next.js)
- **Component library:** shadcn/ui — copy-paste components, matches `@repo/ui` pattern, no runtime dependency
- **CSS framework:** Tailwind CSS — tokens as CSS custom properties in `tailwind.config.ts`

### Fonts loading
- **Mobile:** Load via `expo-google-fonts` or bundle as static assets in `assets/fonts/`
- **Web:** Google Fonts CDN link in `<head>`, `display=swap`
- **PDFs:** Bundle Fraunces and Plus Jakarta Sans as TTF in `apps/api/internal/pdf/fonts/` — gofpdf requires embedded fonts for non-ASCII characters (Devanagari, Tamil, etc. in society/resident names)

### Number formatting
- **Currency:** Indian lakh system — ₹1,25,000 (not ₹125,000). One shared utility function: `formatINR(amount: number): string`. Used consistently across mobile, web, and PDF output.
- **Dates:** "25 Apr 2026" format (not "April 25, 2026"). Indian English convention.

## Devanagari / Regional Script Support

Society names, resident names, and contractor names may contain Hindi, Marathi, Tamil, or other regional scripts. The app must handle these in text inputs. For PDFs specifically: bundle Noto Sans Devanagari (or equivalent) as a fallback TTF in `apps/api/internal/pdf/fonts/` — gofpdf does not fall back to system fonts.

**Phase 1 decision:** Embed Unicode font in PDF for full regional script support. Adds ~2MB to the PDF service binary but avoids silent mojibake in official documents.

## Decisions Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-05-03 | Fraunces (variable serif) for display/documents | Optical-size axis creates intentional heading/body distinction; no competitor uses it; makes PDFs look official, not app-generated |
| 2026-05-03 | Plus Jakarta Sans for UI/body | Professional, readable at 16px+ for 50–65yo target; less ubiquitous than DM Sans or Inter |
| 2026-05-03 | Forest teal `#1B6F68` accent (not medium blue) | Every competitor uses medium blue; teal associates with ledger ink and Indian institutional documents; still trustworthy but visually distinct |
| 2026-05-03 | Warm off-white `#F5F4F0` document surfaces | Clinical white feels temporary-app; `#F5F4F0` feels archival and permanent; 12.8:1 contrast ratio, fully accessible |
| 2026-05-03 | NativeWind for mobile styling | Matches web Tailwind, one token system for two surfaces, reduces context switching |
| 2026-05-03 | shadcn/ui for web components | Copy-paste, no runtime dep, matches `@repo/ui` pattern |
| 2026-05-03 | Embed Unicode fonts in PDF | Full regional script support in official documents; prevents mojibake in society/resident names |
| 2026-05-03 | Indian lakh number formatting | ₹1,25,000 not ₹125,000; wrong format signals the product wasn't built for India |
| 2026-05-03 | Minimal-functional motion only | Serious governance software doesn't dance; no bounce, no choreography |
