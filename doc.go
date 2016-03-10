// docgo is a [Go](http://golang.org) implementation of [Jeremy Ashkenas]
// (http://github.com/jashkenas)'s [docco] (http://jashkenas.github.com/docco/),
// a literate-programming-style documentation generator.  Running docgo on your
// Go source files produces HTML with comments and code side-by-side.
//
// Comments are processed by [Markdown]
// (http://daringfireball.net/projects/markdown) using [Russ Ross]
// (http://github.com/russross)'s [BlackFriday]
// (http://github.com/russross/blackfriday) library, and code is
// syntax-highlighted using [litebrite](http://dhconnelly.github.com/litebrite),
// a Go syntax highlighting library.
//
// The source is available on [GitHub](http://github.com/dhconnelly/docgo).
//
// With a recent Go weekly build (I'm using `weekly.2012-2-07`), you can get,
// install, and run docgo by doing the following at a command line:
//
// `go get github.com/dhconnelly/docgo`<br>
// `go install github.com/dhconnelly/docgo`<br>
// `docgo file.go`
//
// This will create `file.html` in the current directory.
//
// There are two command-line flags:
//
// - `resdir`: a path to the directory containing the CSS styles and HTML
//   templates (this is usually the docgo source directory)
// - `outdir`: the directory to which the generated HTML should be writen

// docgo is copyright 2012 Daniel Connelly.  All rights reserved.  Use of
// this source code is governed by a BSD-style license that can be found in
// the `LICENSE` file.

// ## Imports and globals

package main

import (
	"bytes"
	"flag"
	"go/build"
	"io"
	"io/ioutil"
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
	templ       *template.Template                // html template for generated docs
	match       = regexp.MustCompile(`^\s*//\s?`) // pattern for extracted comments
	sep         = "/*[docgoseparator]*/"          // replacement for comment groups
	unsep       = regexp.MustCompile(`<div class="comment">/\*\[docgoseparator\]\*/</div>`)
	outdir      = flag.String("outdir", ".", "output directory for html & css")
	resdir      = flag.String("resdir", "", "directory containing CSS and templates")
	csspath     = flag.String("csspath", "", "relative path to CSS file, for use with the <link> element")
	cssfilename = "doc.css"
	tplfilename = "doc.templ"
	pkg         = "github.com/christophberger/docgo" // for locating the resources if not specified
)

// ## Generating documentation

type docs struct {
	Filename string
	Sections []*section
	CssPath  string
}

type section struct {
	Doc  string
	Code string
}

// Extract comments from source code, pass them through markdown, highlight the
// code, and render to a string.
func GenerateDocs(title, src string) string {
	sections := extractSections(src)
	highlightCode(sections)
	markdownComments(sections)

	var b bytes.Buffer
	cleanCssPath := ""
	if len(*csspath) > 0 {
		cleanCssPath = path.Clean(*csspath) + string(os.PathSeparator)
	}
	err := templ.Execute(&b, docs{title, sections, cleanCssPath + cssfilename})
	if err != nil {
		panic(err.Error())
	}
	return b.String()
}

// ## Processing sections

// Split the source into sections, where each section contains a comment group
// (consecutive leading-line // comments) and the code that follows that group.
func extractSections(source string) []*section {
	sections := make([]*section, 0)
	current := new(section)

	for _, line := range strings.Split(source, "\n") {
		// When a candidate comment line is found, add it to the current
		// comment group (or create a new section if code has already been
		// added to the current section).
		if match.FindString(line) != "" {
			if current.Code != "" {
				sections = append(sections, current)
				current = new(section)
			}
			// Strip out the comment delimiters
			current.Doc += match.ReplaceAllString(line, "") + "\n"
		} else {
			current.Code += line + "\n"
		}
	}

	return append(sections, current)
}

// Apply markdown to each section's documentation.
func markdownComments(sections []*section) {
	for _, section := range sections {
		// IMHO BlackFriday should use a string interface, since it
		// operates on text (not arbitrary binary) data...
		section.Doc = string(blackfriday.MarkdownCommon([]byte(section.Doc)))
	}
}

// Apply syntax highlighting to each section's code.
func highlightCode(sections []*section) {
	// Rejoin the source code fragments, using sep as delimiter
	segments := make([]string, 0)
	for _, section := range sections {
		segments = append(segments, section.Code)
	}
	code := strings.Join(segments, sep)

	// Highlight the joined source
	h := litebrite.Highlighter{"operator", "ident", "literal", "keyword", "comment"}
	hlcode := h.Highlight(code)

	// Collect the code between subsequent `unsep`s
	matches := append(unsep.FindAllStringIndex(hlcode, -1), []int{len(hlcode), 0})
	lastend := 0
	for i, match := range matches {
		sections[i].Code = hlcode[lastend:match[0]]
		lastend = match[1]
	}
}

// ## Setup and running

// Locate the HTML template and CSS.
func findResources() string {
	if *resdir != "" {
		return *resdir
	}

	// find the path to the package root to locate the resource files
	p, err := build.Default.Import(pkg, "", build.FindOnly)
	if err != nil {
		panic(err.Error())
	}
	return p.Dir
}

// Load the HTML template
func loadTemplate(path string) {
	templ = template.Must(template.ParseFiles(path + string(os.PathSeparator) + tplfilename))
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

// copyCssFile() copies
func copyCssFile() {
	// Copy only if dest path != source path
	src := path.Clean(*resdir)
	dst := path.Clean(*outdir + *csspath)
	if dst != src {
		err := copyFile(dst+string(os.PathSeparator)+cssfilename, src+string(os.PathSeparator)+cssfilename)
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
	outname := filepath.Join(*outdir, name[:len(name)-2]) + "html"
	docs := GenerateDocs(name, string(src))
	err = ioutil.WriteFile(outname, []byte(docs), 0666)
	if err != nil {
		panic(err.Error())
	}
	copyCssFile()
}

func main() {
	flag.Parse()
	loadTemplate(findResources())
	for _, filename := range flag.Args() {
		processFile(filename)
	}
}
