package store

// PlayableFileIDs is integration-tested against a live Postgres because
// it joins the host's public.media_files table. Pure-Go unit tests are
// not meaningful here. See README for the operator-side grants required
// to run that.
