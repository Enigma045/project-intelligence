# Phase 3 notes

Review of Phase 2 output: graph and resolver work. Problems found and fixed here:

1. 128 unresolved calls were noise: method calls on locals, params and package vars (b.String, t.Fatalf, fs.Parse, need, show, reClass.Find...).
   Go extractor now tags calls whose root is a local: Target = local or local:<type> (from params, receivers, var decls, &T{}, new(T)).
   Resolver uses the declared type to resolve receiver.method precisely (status resolved) or to mark external.
2. New edge statuses:
   value    = call on a value (local/param/package var), target not statically known
   dynamic  = dotted call, no project candidate and receiver type unknown
   unresolved is now reserved for real misses (plain names, or qualified calls into a project package).
3. Incremental engine: ExtractorVersion in meta forces a one-time re-extract when extractors change.
   scan now reports per-file symbol diffs and an IMPACT list (dependents of signature-changed or removed symbols).

Known limits / next:
- Return-type inference (x := pkg.New()) is not done yet, so those receivers stay value.
- Go imports still fan out to every file of the imported package; render per package in progress.md (Phase 5).
- Phase 4: HTML/CSS/SQL extractors and DOM links.
