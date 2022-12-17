# fltk-editor-go

This is an experiment of making text editor/processor using go-fltk binding.

# Build

Same as gnote, however as of now no sqlite in use thus sqlite tags is not used

```
go build -ldflags="-s -w -H=windowsgui" --tags "json1 fts5 secure_delete"
```

# Windows dll dependencies

Some dlls needs to be bundled. See the build script `maketar.sh` for details

# Why

fltk is less system resources than gtk3 and I can not load around 60Mb of text file to gnote (using gtk3 golang binding). However this editor is as fast as a blink!

I implemented the text processing feature the same way as gnote so we can search/replace/run script to process the text data. Even better then gnote, we can load a script in as a program and process the current text data. Maybe I will implement it in gnote one day.

Missing feature is syntax highlighting - maybe a TODO.