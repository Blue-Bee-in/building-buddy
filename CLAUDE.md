# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Monorepo Structure

This is a **pnpm + Turborepo** monorepo with three apps and three shared packages:

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

For a complete reference of all major user journeys (society onboarding, resident onboarding, billing, visitor management, delivery, domestic help, complaints, and emergency SOS), refer to [`docs/flow.md`](./docs/flow.md).

## Architecture Notes

### API
- Entry point: `apps/api/cmd/server/main.go`
- Uses `chi` for HTTP routing with CORS via `go-chi/cors`
- Database: PostgreSQL via `pgx/v5`; migrations managed by `goose`; query builder is `bob`
- The `air` tool watches for file changes in dev (config likely at `apps/api/.air.toml`)

### Web (admin)
- Package name is `admin` internally
- Next.js App Router; pages live in `apps/web/app/`
- Runs on port 3000 in dev
- `next typegen` must run before `tsc` (it's part of the `check-types` script)

### Mobile
- File-based routing via `expo-router`; route files are in `apps/mobile-app/app/`
- Tab navigation defined under `app/(tabs)/`
- Uses `react-native-reanimated` and `react-native-gesture-handler`
- `@/` path alias maps to the project root

### Shared UI (`@repo/ui`)
- Components exported as `@repo/ui/<name>` (e.g. `@repo/ui/button`)
- Each component is a single `.tsx` file under `packages/ui/src/`
- Only consumed by Next.js apps (not the mobile app)
