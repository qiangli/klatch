---
id: 69c61bc5fbe4
kind: enhancement
title: Swap Klatch to a pure-Go SQLite driver (CGO_ENABLED=0, enables Windows)
seq: 2827
status: done
priority: p1
labels:
    - build
    - windows
    - sqlite
created: 2026-11-07T03:13:49.370663Z
closed: 2026-11-07T03:16:45.103492Z
---

Problem: net/server/internal/store/alloc.go opens the store through gorm.io/driver/sqlite, which uses the cgo driver github.com/mattn/go-sqlite3, so every Klatch build needs CGO_ENABLED=1 and a C compiler. Klatch v1.2.6 therefore ships linux/darwin only and Windows hosts keep running the old Klatch container (bashy plan S563 F2). bashy itself is pure Go (modernc.org/sqlite).
Change: replace gorm.io/driver/sqlite with the pure-Go gorm driver github.com/glebarez/sqlite (modernc.org/sqlite underneath) at the single call site. Translate the DSN: mattn's '?_busy_timeout=5000' is NOT understood by modernc and would be silently ignored, reintroducing the SQLite lock bug fixed in c5cac3f9. Use '?_pragma=busy_timeout(5000)' and keep any other pragma semantics identical. go mod tidy so mattn/go-sqlite3 leaves the module graph.
Compatibility: same SQLite file format. Prove that an existing databag.db opens and migrates (AutoMigrate) unchanged.
Verification: CGO_ENABLED=0 go build ./... for linux, darwin and windows; the store tests and the team-chat e2e test (concurrent first visit, lock) pass with CGO_ENABLED=0; go list -deps shows no mattn/go-sqlite3.
Follow-up (bashy repo, builds/klatch): flip CGO_ENABLED to 0, add a windows/amd64 leg with its no-egress gate shape, then cut Klatch v1.2.7 and re-pin.
