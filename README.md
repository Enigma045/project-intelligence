# pidx - Project Intelligence (Phases 0-1)

Commands (flags go BEFORE positional args):

    pidx scan    -root <dir>             incremental index into <dir>/.project-intelligence/index.db
    pidx outline -root <dir> [-json] <relpath>   symbols of one file (hash-verified against disk)
    pidx search  -root <dir> <text>      symbol search by name
    pidx refs    -root <dir> <name>      raw call/DOM/event/API references to a name
    pidx stats   -root <dir>             counts
    pidx extract [-json] <file>          run an extractor without the DB

## Status
- Go: go/ast, provenance=verified.
- JS/TS and Python: heuristic extractors, provenance=inferred. They sit behind the
  extract.Extractor interface (internal/extract/extract.go registry) so Tree-sitter
  adapters can replace them later without touching the store or CLI.
- Unknown types are stored as type_source=unknown, never guessed.
- Not yet implemented: resolver/graph (Phase 2), HTML/CSS/SQL (Phase 4), docs/retrieval (Phase 5), edit engine (Phase 6).

## Known limits of the heuristic extractors
- Multi-line import statements and multi-line parameter lists (JS) are not fully parsed.
- Braces inside regex literals can confuse JS block-end detection.
- Python block ends rely on indentation.
