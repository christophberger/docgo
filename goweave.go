//go:generate go-bindata -o resources.go resources
/*
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

1. Install goweave and its dependencies through go get:

        go get github.com/christophberger/goweave/...

2. If you plan to modify files in the `resources/` folder, install go-bindata...

		go get github.com/jteeuwen/go-bindata

   ...and run `go generate` each time you modify the CSS file or the template file.

3. (Optional) Install the CSS and template files into `~/.config/goweave`:

		goweave -install

4. Run goweave on a Go file with comments:

        goweave mycode.go

5. Open the generated mycode.html in a browser.


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
*/

// ## The code

// ### Imports and globals
//
package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/dhconnelly/litebrite"
	"github.com/russross/blackfriday"
)

var (
	style            string
	templ            *template.Template // html template for generated docs
	commentPtrn      = `^\s*//\s?`
	commentStartPtrn = `^\s*/\*\s?`
	commentEndPtrn   = `\s?\*/\s*$`
	directivePtrn    = `^//go:`
	comment          = regexp.MustCompile(commentPtrn)      // pattern for single-line comments
	commentStart     = regexp.MustCompile(commentStartPtrn) // pattern for /* comment delimiter
	commentEnd       = regexp.MustCompile(commentEndPtrn)   // pattern for */ comment delimiter
	directive        = regexp.MustCompile(directivePtrn)    // pattern for //go: directive, like //go:generate
	allCommentDelims = regexp.MustCompile(commentPtrn + "|" + commentStartPtrn + "|" + commentEndPtrn)
	outdir           = flag.String("outdir", ".", "output directory for html & css")
	resdir           = flag.String("resdir", "", "directory containing CSS and templates")
	csspath          = flag.String("csspath", "", "relative path to CSS file, for use with the <link> element")
	md               = flag.Bool("md", false, "generate Markdown document (default: HTML)")
	bare             = flag.Bool("bare", false, "generate the HTML body only")
	inline           = flag.Bool("inline", false, "generate inline CSS")
	installResources = flag.Bool("install", false, "install resource files into .config/goweave")
	intro            = flag.Bool("intro", false, "Only process the first comment section (that should contain some intro text).")
	cssfilename      = "goweave.css"
	tplfilename      = "goweave.templ"
	configDir        = filepath.Join(getHomeDir(), ".config", "goweave")
	resourcedir      = "" // resource directory as determined by findResources()
)

// ### Generating documentation
//
type docs struct {
	Filename  string
	Sections  []*section
	CssPath   string
	Style     string
	Full      bool
	InlineCSS bool
}

type section struct {
	Doc  string
	Code string
}

// Extract comments from source code, pass them through markdown, highlight the
// code, and render to a string.
func generateDocs(title, src string) (result string) {
	sections := extractSections(src)

	if !*md {
		highlightCode(sections)
		markdownComments(sections)
		var b bytes.Buffer
		cleanCssPath := ""
		if len(*csspath) > 0 {
			cleanCssPath = path.Clean(*csspath) + string(os.PathSeparator)
		}
		// Now apply the template.
		err := templ.Execute(&b, docs{title, sections, cleanCssPath + cssfilename, style, !*bare, *inline})
		if err != nil {
			panic(err.Error())
		}
		result = b.String()
	} else {
		if !*intro { // Skip this if rendering the intro text only, to avoid an empty code block in the output.
			markdownCode(sections)
		}
		result = joinSections(sections)
	}
	return result
}

// ### Processing sections
//
// Determine if the current line belongs to a comment region. A comment region
// is either a comment line (starting with `//`) or a `/*...*/` multi-line comment.
func commentFinder() func(string) bool {
	commentSectionInProgress := false
	return func(line string) bool {
		if comment.FindString(line) != "" {
			// "//" Comment line found.
			return true
		}
		// If the current line is at the start `/*` of a multi-line comment,
		// set a flag to remember we're within a multi-line comment.
		if commentStart.FindString(line) != "" {
			commentSectionInProgress = true
			return true
		}
		// At the end `*/` of a multi-line comment, clear the flag.
		if commentEnd.FindString(line) != "" {
			commentSectionInProgress = false
			return true
		}
		// The current line is within a `/*...*/` section.
		if commentSectionInProgress {
			return true
		}
		// Anything else is not a comment region.
		return false
	}
}

// isDirective returns true if the input argument is a Go directive.
func isDirective(line string) bool {
	if directive.FindString(line) != "" {
		return true
	}
	return false
}

// Split the source into sections, where each section contains a comment group
// and the code that follows that group.
func extractSections(source string) []*section {
	var sections []*section
	current := new(section)
	isInComment := commentFinder()

	for _, line := range strings.Split(source, "\n") {
		// Skip the line if it is a Go directive like //go:generate
		if isDirective(line) {
			continue
		}
		// Determine if the line belongs to a comment.
		if isInComment(line) {
			// If currently in a Code group, switch to a new section.
			if current.Code != "" {
				sections = append(sections, current)
				current = new(section)
			}
			// Strip out any comment delimiter and add the line to the
			// Doc group.
			current.Doc += allCommentDelims.ReplaceAllString(line, "") + "\n"

		} else {
			// Stop here if only the intro text shall be rendered.
			if *intro {
				break
			}
			// Add the current line to the Code group.
			current.Code += line + "\n"
		}
	}
	return append(sections, current)
}

// Join sections into a single string.
func joinSections(sections []*section) (res string) {
	for _, s := range sections {
		res += s.Doc
		res += s.Code
	}
	return res
}

// markdownString applies markdown to the input string, using the
// commonHtmlFlags and commonExtensions as defined in blackfriday/markdown.go,
// plus HTML_HREF_TARGET_BLANK.
func markdownString(input string) string {
	const (
		htmlFlags = 0 |
			blackfriday.HTML_USE_XHTML |
			blackfriday.HTML_USE_SMARTYPANTS |
			blackfriday.HTML_SMARTYPANTS_FRACTIONS |
			blackfriday.HTML_SMARTYPANTS_DASHES |
			blackfriday.HTML_HREF_TARGET_BLANK

		extensions = 0 |
			blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
			blackfriday.EXTENSION_TABLES |
			blackfriday.EXTENSION_FENCED_CODE |
			blackfriday.EXTENSION_AUTOLINK |
			blackfriday.EXTENSION_STRIKETHROUGH |
			blackfriday.EXTENSION_SPACE_HEADERS |
			blackfriday.EXTENSION_HEADER_IDS |
			blackfriday.EXTENSION_BACKSLASH_LINE_BREAK |
			blackfriday.EXTENSION_DEFINITION_LISTS
	)
	renderer := blackfriday.HtmlRenderer(htmlFlags, "", "")
	return string(blackfriday.MarkdownOptions([]byte(input), renderer,
		blackfriday.Options{Extensions: extensions}))
}

// Apply markdown to each section's documentation.
func markdownComments(sections []*section) {
	for _, section := range sections {
		// MarkdownCommon() enables a couple of common Markdown extensions, like
		// Smartypants, tables, fenced code blocks, and more.
		section.Doc = markdownString(section.Doc)
	}
}

// litebrite eats leading whitespace when fed with code snippets.
// To address this, splitLeadingWs splits the code into leading whitespace
// and the rest, to be re-joined after highlighting.
func splitLeadingWs(s string) (string, string) {
	code := strings.TrimLeft(s, "\t ")
	return s[:strings.Index(s, code)], code
}

// Apply syntax highlighting to each section's code.
func highlightCode(sections []*section) {
	h := litebrite.Highlighter{
		OperatorClass: "operator",
		IdentClass:    "ident",
		LiteralClass:  "literal",
		KeywordClass:  "keyword",
		CommentClass:  "comment",
	}
	for i := range sections {
		s := sections[i].Code
		if strings.TrimSpace(strings.Trim(s, "\n")) != "" {
			ws, code := splitLeadingWs(s)
			sections[i].Code = ws + h.Highlight(code)
		} else {
			sections[i].Code = "" // make empty Code *really* empty
		}
	}
}

// Put the code into Markdown code fences
func markdownCode(sections []*section) {
	for i := range sections {
		if sections[i].Code != "\n" {
			sections[i].Code = "\n```go\n" + sections[i].Code + "```\n"
		}
	}
}

// ### Setup and running
//
// Locate the HTML template and CSS.
func findResources() string {
	// If a custom resource dir is given, use that.
	if *resdir != "" {
		return *resdir
	}

	// If there is a "goweave" directory in the current path,
	// and if it contains the css and templ files, use that.
	path := filepath.Join("goweave", "resources")
	res, err := os.Open(path)
	if err == nil {
		_ = res.Close() // An error here is harmless, as we only checked for existence.
		res, err = os.Open(filepath.Join(path, cssfilename))
		if err == nil {
			_ = res.Close() // Same here.
			return path
		}
	}

	// Else try to use the files in $HOME/.config/goweave.
	path = filepath.Join(configDir, "resources")
	cssFile, err := os.Open(filepath.Join(path, cssfilename))
	if err == nil {
		_ = cssFile.Close()
		return path
	}

	// If none of the above was successful, install the resource files from
	// the binary (under "resources") into ./goweave.
	if install("goweave") != nil {
		log.Fatal("Unable to install the resource files into './goweave'.")
	}
	return filepath.Join("goweave", "resources")
}

// Load the HTML template.
// Load the CSS if it shall be inlined.
func loadResources(path string) {
	if *inline {
		data, err := ioutil.ReadFile(filepath.Join(path, "goweave.css"))
		if err != nil {
			panic(err.Error())
		}
		style = string(data)
	}
	templ = template.Must(template.ParseFiles(filepath.Join(path, tplfilename)))
}

// copyFile copies the contents of src to dst atomically.
// Copied from github.com/pkg/fileutils/copy.go.
// (c) Dave Cheney - see LICENSE_CopyFile.txt.
func copyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := ioutil.TempFile(filepath.Dir(dst), "copyfile")
	if err != nil {
		return err
	}
	_, err = io.Copy(tmp, in)
	if err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	const perm = 0644
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// copyCssFile() copies the CSS file to the destination.
// Use -csspath=<path> to specify a relative destination path, e.g.:
// goweave -csspath=css ...
func copyCssFile() {
	// Copy only if dest path != source path
	src := filepath.Join(resourcedir, cssfilename)
	dst := filepath.Join(*outdir, *csspath)

	if os.Chdir(dst) != nil {
		err := os.MkdirAll(dst, os.ModeDir)
		if err != nil {
			panic(err.Error())
		}
		err = os.Chmod(dst, 0744)
		if err != nil {
			panic(err.Error())
		}
	}
	dst = filepath.Join(dst, cssfilename)
	if dst != src {
		err := copyFile(dst, src)
		if err != nil {
			panic(err.Error())
		}
	}
}

// Generate documentation for a source file.
func processFile(filename string) {
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err.Error())
	}
	name := filepath.Base(filename)
	ext := "html"
	if *md {
		ext = "md"
	}
	outname := filepath.Join(*outdir, name[:len(name)-2]) + ext
	docs := generateDocs(name, string(src))
	err = ioutil.WriteFile(outname, []byte(docs), 0666)
	if err != nil {
		panic(err.Error())
	}
	if !*inline {
		copyCssFile()
	}
}

// getHomeDir finds the user's home directory in an OS-independent way.
// "OS-independent" means compatible with most Unix-like operating systems as well as with Microsoft Windows(TM).\
// Credits for the OS-independent approach used here go to http://stackoverflow.com/a/7922977.
// (os.User is not an option here. It relies on CGO and thus prevents cross compiling.)
func getHomeDir() string {
	home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		home = os.Getenv("HOME")
	}
	return home
}

// install writes the CSS and Template files into ~/.config/goweave.
// The source files are stored in the binary via go-bindata.
// If you change the original CSS or Template files in the git/go workspace,
// run go generate.
func install(targetDir string) error {
	return RestoreAssets(targetDir, "resources")
}

func main() {
	flag.Parse()
	if *installResources {
		if install(configDir) != nil {
			log.Fatal("Unable to install the resource files into '" + configDir + "'.")
		}
		return
	}
	resourcedir = findResources()
	loadResources(resourcedir)
	for _, filename := range flag.Args() {
		processFile(filename)
	}
}
