# Rules for AI agents

obstenku is collision detection and response for axis-aligned rectangles in Go games.
Module `github.com/shpaker/obstenku`, one package, no dependencies.

## Architecture

```
obstenku.go      package docs, eps, Vec2, Response, Filter
geometry.go      Rect, Sweep, SegmentRect: the math of one pair
world.go         World: items, Check and Move, queries
internal/grid/   the spatial hash behind World
```

Architecture and the right abstractions come first.

**Scope: rectangles that meet.** obstenku finds where boxes meet and where a moving one
stops. Speeds, forces, input, drawing, sound and what a hit means belong to the game:
the game decides how a pair meets with a `Filter` and acts on the collisions it gets
back.

- Pure geometry: no engine, no IO, no platform code (depguard checks). Works the same on
  every platform Go builds for.
- No global state, no wall clock, no randomness: the same calls give the same results.
- The public API lives in the root package (in Go a package is a directory); helpers
  that are not API go to `internal/`. A few files, grouped by area, not one per type.
- All the math of a pair lives in `geometry.go`; `World` only picks the candidates and
  applies the responses.
- World, filters, responses and queries do not depend on the shape: a new shape is a
  new pair test, not a new API. The normal is a vector, not one of four directions.
- Touching is not overlapping: boxes closer than `eps` only touch. A touch lies exactly
  on the face, so rounding never sinks a box into a wall.
- The promise: a moving box never passes through an item that stops it and never goes
  deeper into one it started inside; where a move ends does not depend on the order of
  adding. `FuzzNoTunneling` and `FuzzSameInAnyOrder` check it; a change must keep them
  green.

## Principles

- One type — one responsibility.
- No deprecated APIs (e.g. `rand.Seed`).
- Slices are returned by appending to a `dst` the caller passes, so a game can reuse
  them every tick.

## Naming

- Constructors `New*`; getters without `Get` (`Rect`, `Len`), setters `Set*`.
- No abbreviations.
- Short clear names.
- Fields as private as possible, grouped by kind.

## Style

- Standard Go conventions; short variable declarations where they fit.
- Comments and docs in English, plain words.
- Document every exported identifier.
- Tests as simple and readable as possible. `example_test.go` mirrors the README's
  quick start, so the README never drifts from the code.

## Required after changes

- `just fmt`, `just lint`, `just lint-wasm`, `just test`, `just fuzz`; the linter stays
  clean for both targets (desktop and js/wasm).
- Find and update every use of a changed identifier; update the tests.
- Update README when the API changes: what a game developer sees — short and to the
  point.

## Git

- Any writing git operation (commit, push, merge, rebase, reset, tag and the like) only with
  the user's explicit confirmation.
- Commits are made as the user only.
- No mention of AI or neural networks (`Co-Authored-By`, `Generated with ...`, `claude/`,
  `ai/` and the like) in commits, their metadata, branch names and PRs.
- PR and release descriptions as short as possible; the only formatting is a bulleted list,
  no headings or emphasis.
- Conventional Commits: `feat`, `fix`, `perf`, `refactor` raise the minor version,
  `type!:` or a `BREAKING CHANGE:` footer the major (while at v0 — the minor); `docs`,
  `chore`, `ci`, `test`, `style`, `build` make no release.
- Release tags (`vMAJOR.MINOR.0`, as Go modules require) are set by CI
  (`.github/workflows/version.yml`), run by hand on main with a green CI: changes pile up,
  a release goes out when ready; it tags and publishes the release with its changelog.
  Never tag by hand.
