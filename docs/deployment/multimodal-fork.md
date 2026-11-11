# Multimodal deployment branch

`production` is based on the deployed upstream revision
`f17ee8a8da901bd658a4cb25f40b5fd41adbfdd7`. The fork's `main` branch is retained
for upstream tracking; update production deliberately after validation.

This repair:

- Lets `read_file` inspect knowledge images through existing document/tag/tenant
  authorization and an explicit resource binding to the same document.
- Returns authorized image bytes using the existing image-result preparation path.
- Honors an explicit image-vector opt-in for scanned PDF pages while preserving
  the implicit default skip.
- Preserves source locators on newly created image-vector chunks.

Focused regression check:

```sh
go test -p 1 ./internal/agent/tools ./internal/application/service \
  -run 'TestKnowledgeImage|TestReadFile|TestReadArtifact|TestIndexImageVector|TestParentReadsOnlyDeliveredArtifacts' \
  -count=1
```

The deployed server uses the original `anydoc` build tag and runtime base image;
no application dependency or database schema was added. Existing documents are
not automatically reprocessed. Existing image vectors need selective backfill
through the native indexer when desired; retain document and vector backups first.
Keep deployment credentials, storage objects and one-off document maintenance
scripts outside this public fork.
