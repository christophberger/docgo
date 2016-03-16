// # goweave
//
// goweave creates an HTML file from a Go source file, rendering comments and
// code side-by-side. Comments can use Markdown formatting as accepted by the
// MarkdownCommon() method of the BlackFriday library; see below for details.
//
// Basically, goweave is meant to be the "weave" part of Literate Programming for Go.
//
// Options:
//
// * `-resdir=<dir>`: Resource directory, defaults to the go get directory of goweave.
// * `-outdir=<dir>`: Output directory. Defaults to the current directory.
// * `-csspath=<path>`: Output path for the CSS file.
// * `-inline`: Include the CSS into the HTML file.
// * `-md`: Generate Markdown output rather than HTML.
// * `-bare`: Only generate the body part of the HTML document.
//
// goweave is based on the wonderful [docgo](https://github.com/dhconnelly/docgo)
// project by Daniel Connelly. Although I shuffled much of the code
// around, added new code, removed some, and finally ended up with substantial
// changes to the resulting behavior, docgo saved me a lot--a LOT!--of time as
// it had all the groundworks already done for me.
//
// docgo in turn is a [Go](http://golang.org) implementation of [Jeremy Ashkenas]
// (http://github.com/jashkenas)'s [docco] (http://jashkenas.github.com/docco/),
// a literate-programming-style documentation generator.
//
// Comments are processed by [Markdown]
// (http://daringfireball.net/projects/markdown) using [Russ Ross]
// (http://github.com/russross)'s [BlackFriday]
// (http://github.com/russross/blackfriday) library, and code is
// syntax-highlighted using [litebrite](http://dhconnelly.github.com/litebrite),
// a Go syntax highlighting library.
//
// Optionally you can generate a Markdown document instead of HTML. In this case,
// you need to provide your own CSS that matches the output of your Markdown
// renderer.
// Also ensure your Markdown renderer is able to process "````````go" tags correctly.
//
//
// goweave is copyright 2016 by Christoph Berger. All rights reserved.
// This source code is governed by a BSD-style license that can be found in
// the `LICENSE.txt` file.
//
// License files for godoc, litebrite, blackfriday, and the CopyFile function from
// github.com/pkg/fileutils/copy.go:
//
// * LICENSE_godoc.md
// * LICENSE_litebrite.md
// * LICENSE_blackfriday.txt
// * LICENSE_CopyFile.txt

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
	style       string
	templ       *template.Template                // html template for generated docs
	match       = regexp.MustCompile(`^\s*//\s?`) // pattern for extracted comments
	outdir      = flag.String("outdir", ".", "output directory for html & css")
	resdir      = flag.String("resdir", ".", "directory containing CSS and templates")
	csspath     = flag.String("csspath", "", "relative path to CSS file, for use with the <link> element")
	md          = flag.Bool("md", false, "generate Markdown document (default: HTML)")
	bare        = flag.Bool("bare", false, "generate the HTML body only")
	inline      = flag.Bool("inline", false, "generate inline CSS")
	cssfilename = "goweave.css"
	tplfilename = "goweave.templ"
	pkg         = "github.com/christophberger/goweave" // for locating the resources if not specified
)

// ## Generating documentation

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
func GenerateDocs(title, src string) (result string) {
	sections := extractSections(src)

	if !*md {
		highlightCode(sections)
		markdownComments(sections)
		var b bytes.Buffer
		cleanCssPath := ""
		if len(*csspath) > 0 {
			cleanCssPath = path.Clean(*csspath) + string(os.PathSeparator)
		}
		err := templ.Execute(&b, docs{title, sections, cleanCssPath + cssfilename, style, !*bare, *inline})
		if err != nil {
			panic(err.Error())
		}
		result = b.String()
	} else {
		markdownCode(sections)
		result = joinSections(sections)
	}
	return result
}

// ## Processing sections

// Split the source into sections, where each section contains a comment group
// (consecutive leading-line // comments) and the code that follows that group.
func extractSections(source string) []*section {
	sections := make([]*section, 0)
	current := new(section)

	for _, line := range strings.Split(source, "\n") {
		// When a candidate comment line is found, first test if we're currently
		// in a Code group. If so, switch to a new section. Add the comment line
		// to the section's Doc group.
		if match.FindString(line) != "" {
			if current.Code != "" {
				sections = append(sections, current)
				current = new(section)
			}
			// Strip out the comment delimiters and add the line to the
			// current doc section.
			current.Doc += match.ReplaceAllString(line, "") + "\n"
		} else {
			// No comment line found, so add the current line to the Code group.
			current.Code += line + "\n"
		}
	}

	return append(sections, current)
}

// Join sections into a single string
func joinSections(sections []*section) (res string) {
	for _, s := range sections {
		res += s.Doc
		res += s.Code
	}
	return res
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
	h := litebrite.Highlighter{"operator", "ident", "literal", "keyword", "comment"}
	for i := range sections {
		if strings.Trim(strings.TrimSpace(sections[i].Code), "\n") != "" {
			sections[i].Code = h.Highlight(sections[i].Code)
		} else {
			sections[i].Code = "" // make empty Code *really* empty
		}
	}
}

// Put the code into Markdown code fences
func markdownCode(sections []*section) {
	for i := range sections {
		sections[i].Code = "\n```go\n" + sections[i].Code + "```\n"
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

// Load the HTML template.
// Load the CSS if it shall be inlined.
func loadResources(path string) {
	if *inline {
		data, err := ioutil.ReadFile(path + "/goweave.css")
		if err != nil {
			panic(err.Error())
		}
		style = string(data)
	}
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

// copyCssFile() copies the CSS file to the destination.
// Use -csspath=<path> to specify a relative destination path, e.g.:
// goweave -csspath=css ...
func copyCssFile() {
	// Copy only if dest path != source path
	ps := string(os.PathSeparator)
	src := path.Clean(*resdir + ps + cssfilename)
	dst := path.Clean(*outdir + ps + *csspath)

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
	dst += ps + cssfilename
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
	docs := GenerateDocs(name, string(src))
	err = ioutil.WriteFile(outname, []byte(docs), 0666)
	if err != nil {
		panic(err.Error())
	}
	if !*inline {
		copyCssFile()
	}
}

func main() {
	flag.Parse()
	loadResources(findResources())
	for _, filename := range flag.Args() {
		processFile(filename)
	}
}
