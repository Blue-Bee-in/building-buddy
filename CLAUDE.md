# CLAUDE.md

Guidance for Claude Code (claude.ai/code) working in this repo.

## Monorepo Structure

**pnpm + Turborepo** monorepo. Three apps, three shared packages:

- `apps/api` — Go REST API (`chi` router, `pgx` for Postgres, `goose` migrations, `bob` ORM, `air` for hot-reload)
- `apps/web` — Next.js 16 admin frontend (React 19, TypeScript, uses `@repo/ui`)
- `apps/mobile-app` — Expo/React Native app (expo-router file-based routing, targets iOS/Android)
- `apps/docs` — Next.js docs site
- `packages/ui` — Shared React component library consumed by `web` and `docs`
- `packages/eslint-config` — Shared ESLint config
- `packages/typescript-config` — Shared `tsconfig.json` base configs

## Commands

### Root (all apps via Turborepo)
```sh
pnpm dev          # Start all apps in dev mode
pnpm build        # Build all apps
pnpm lint         # Lint all apps/packages
pnpm check-types  # Type-check all apps/packages
pnpm format       # Prettier format all TS/TSX/MD files
```

### Filtered (single app)
```sh
pnpm turbo dev --filter=web
pnpm turbo dev --filter=mobile-app
pnpm turbo dev --filter=api
pnpm turbo build --filter=web
```

### API (Go — run from `apps/api`)
```sh
air                        # Dev server with hot-reload
go build -o bin/server     # Production build
staticcheck ./...          # Static analysis (check-types)
go test ./...              # Run tests
go test ./path/to/pkg      # Run single package tests
```

### Mobile app (run from `apps/mobile-app`)
```sh
expo start --ios           # Start on iOS simulator (default dev)
expo start --android       # Start on Android
eas build --platform android --local  # Production Android build
```

### Generate a new UI component
```sh
cd packages/ui && pnpm generate:component
```

## Application Flows

All major user journeys (society onboarding, resident onboarding, billing, visitor management, delivery, domestic help, complaints, emergency SOS): [`docs/flow.md`](./docs/flow.md).

## Architecture Notes

### API
- Entry: `apps/api/cmd/server/main.go`
- `chi` HTTP routing, CORS via `go-chi/cors`
- DB: PostgreSQL via `pgx/v5`; migrations: `goose`; query builder: `bob`
- `air` watches file changes in dev (config: `apps/api/.air.toml`)

### Web (admin)
- Package name: `admin` internally
- Next.js App Router; pages in `apps/web/app/`
- Port 3000 in dev
- `next typegen` must run before `tsc` (part of `check-types` script)

### Mobile
- File-based routing via `expo-router`; routes in `apps/mobile-app/app/`
- Tab nav under `app/(tabs)/`
- Uses `react-native-reanimated` and `react-native-gesture-handler`
- `@/` alias maps to project root

### Shared UI (`@repo/ui`)
- Components exported as `@repo/ui/<name>` (e.g. `@repo/ui/button`)
- Each component: single `.tsx` file under `packages/ui/src/`
- Next.js apps only (not mobile)
## gstack (REQUIRED — global install)

**Before doing ANY work, verify gstack is installed:**

```bash
test -d ~/.claude/skills/gstack/bin && echo "GSTACK_OK" || echo "GSTACK_MISSING"
```

If GSTACK_MISSING: STOP. Do not proceed. Tell the user:

> gstack is required for all AI-assisted work in this repo.
> Install it:
> ```bash
> git clone --depth 1 https://github.com/garrytan/gstack.git ~/.claude/skills/gstack
> cd ~/.claude/skills/gstack && ./setup --team
> ```
> Then restart your AI coding tool.

Do not skip skills, ignore gstack errors, or work around missing gstack.

Using gstack skills: After install, skills like /qa, /ship, /review, /investigate,
and /browse are available. Use /browse for all web browsing.
Use ~/.claude/skills/gstack/... for gstack file paths (the global path).
