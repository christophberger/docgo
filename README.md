# goweave

**Generate docs from code like Literate Programming**

## About

goweave creates an HTML file from a Go source file, rendering comments and
code side-by-side. Comments can use Markdown formatting as accepted by the
MarkdownCommon() method of the BlackFriday library; see below for details.

Goweave is meant to work like "weave" in [Literate Programming]
(https://en.wikipedia.org/wiki/Literate_programming). Unlike in Literate Programming,
no "tangle" counterpart is required, as the source document is already a
valid Go source file, ready to be `go install`'ed.

## Use Cases

* Read code and comments side-by-side (if your browser's viewport is wide enough).
* Generate blog articles from a single code file.

## Getting Started

1. Install goweave through go get.

        go get github.com/christophberger/goweave

2. Run goweave on a Go file with comments:

        goweave mycode.go

3. Open the generated mycode.html in a browser.


## Options

* `-install`: Installs resource files into `$HOME/.config/goweave`.
* `-resdir=<dir>`: Resource directory.(1)
* `-outdir=<dir>`: Output directory. Defaults to the current directory.
* `-csspath=<path>`: Output path for the CSS file, relative to the output directory.
  Defaults to the current directory.
* `-bare`: Only generate the body part of the HTML document. (No CSS file references is
  included then, use -inline instead or add the CSS reference manually in your HTML
  header.
* `-inline`: Include the CSS into the HTML file. Does not work with `-bare`.
* `-md`: Generate Markdown output rather than HTML.(2)
* `-intro`: Only process the very first comment (which should be some intro text that
  can be read as-is). Together with -md this comes handy for easily generating a
  README.md from the source.

(1) If -resdir is not given, goweave searches for `goweave/resources` first in the
current dir, then in $HOME/config. If neither succeeds, it automatically installs
the resource files into `./goweave/resources`.

(2) If you generate a Markdown document instead of HTML, you need to provide your
own CSS that matches the output of your Markdown renderer.\
Also ensure your Markdown renderer is able to process "```go" code fences correctly.\
Side-by-side rendering of comments and code does not work in this mode unless you
tweak your CSS and/or your markdown renderer accordingly.


## Notes

### Full-width sections

If a comment is not followed by code but rather by another comment (separated
by an empty line), this comment gets rendered in the center of the document and
without a code column.

This can be useful for creating intro sections or READMEs, or for splitting
long code into separate snippets.


## Origins

goweave is based on the wonderful [docgo](https://github.com/dhconnelly/docgo)
project by [Daniel Connelly](https://github.com/dhconnelly). Although I
shuffled much of the code around, added new code, removed some, and finally
ended up with substantial changes to the resulting behavior, docgo saved me
a lot--a LOT!--of time as it had all the groundworks already done for me.

docgo in turn is a [Go](http://golang.org) implementation of [Jeremy Ashkenas]
(http://github.com/jashkenas)'s [docco] (http://jashkenas.github.com/docco/),
a literate-programming-style documentation generator.

Comments are processed by [Markdown] (http://daringfireball.net/projects/markdown)
using [Russ Ross] (http://github.com/russross)'s [BlackFriday]
(http://github.com/russross/blackfriday) library, and code is
syntax-highlighted using [litebrite](http://dhconnelly.github.com/litebrite),
a Go syntax highlighting library.


## Licenses

This source code is governed by a BSD-style license that can be found in
the `LICENSE.txt` file.

The original docgo code is copyright 2012 by Daniel Connelly. See `LICENSE_godoc`.

See these files for the licenses of litebrite, blackfriday, and the CopyFile function
from github.com/pkg/fileutils/copy.go:

* LICENSE_litebrite.md
* LICENSE_blackfriday.txt
* LICENSE_CopyFile.txt

