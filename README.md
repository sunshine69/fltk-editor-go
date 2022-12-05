# fltk-editor-go

This is an experiment of making text editor/processor using go-fltk binding.

# Build

Same as gnote, however as of now no sqlite in use thus sqlite tags is not used

```
go build -ldflags="-s -w -H=windowsgui" --tags "json1 fts5 secure_delete"
```

# Windows dll dependencies

Some dlls needs to be bundled. See the build script `maketar.sh` for details