package main

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestGenerateDocs(t *testing.T) {
	tests := []struct {
		title string
		src   string
		want  string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		if got := GenerateDocs(tt.title, tt.src); got != tt.want {
			t.Errorf("%q. GenerateDocs() = %v, want %v", tt.title, got, tt.want)
		}
	}
}

func TestCommentFinder(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"// Comment", true},
		{"package main", false},
		{"/* Begin", true},
		{"within", true},
		{"", true},
		{"End */", true},
		{"func test() {", false},
		{"", false},
	}
	isInComment := commentFinder()
	for _, tt := range tests {
		if got := isInComment(tt.line); got != tt.want {
			t.Errorf("%q. commentFinder() = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestExtractSections(t *testing.T) {
	tests := []struct {
		source string
		want   []*section
	}{
		{`// Test comment
// more comment

Test code
More code

// Second comment
  Second code snippet

/* Third comment
In comment section
End of comment */
`,
			[]*section{{`Test comment
more comment
`,
				`
Test code
More code

`},
				{"Second comment\n",
					"  Second code snippet\n\n"},
				{"Third comment\nIn comment section\nEnd of comment\n",
					"\n"},
			},
		},
	}
	for _, tt := range tests {
		if got := extractSections(tt.source); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("extractSections(%v) = %v, want %v", tt.source, spew.Sdump(got), spew.Sdump(tt.want))
		}
	}
}

func TestJoinSections(t *testing.T) {
	tests := []struct {
		sections []*section
		want     string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		if got := joinSections(tt.sections); got != tt.want {
			t.Errorf("joinSections(%v) = %v, want %v", tt.sections, got, tt.want)
		}
	}
}

func TestMarkdownComments(t *testing.T) {
	tests := []struct {
		sections []*section
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		markdownComments(tt.sections)
	}
}

func TestHighlightCode(t *testing.T) {
	tests := []struct {
		sections []*section
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		highlightCode(tt.sections)
	}
}

func TestMarkdownCode(t *testing.T) {
	tests := []struct {
		sections []*section
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		markdownCode(tt.sections)
	}
}

func TestFindResources(t *testing.T) {
	tests := []struct {
		want string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		if got := findResources(); got != tt.want {
			t.Errorf("findResources() = %v, want %v", got, tt.want)
		}
	}
}

func TestCopyFile(t *testing.T) {
	tests := []struct {
		dst     string
		src     string
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		if err := copyFile(tt.dst, tt.src); (err != nil) != tt.wantErr {
			t.Errorf("copyFile(%v, %v) error = %v, wantErr %v", tt.dst, tt.src, err, tt.wantErr)
		}
	}
}

func TestProcessFile(t *testing.T) {
	tests := []struct {
		filename string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		processFile(tt.filename)
	}
}

func TestMain(t *testing.T) {
	tests := []struct {
	}{
	// TODO: Add test cases.
	}
	for range tests {
		main()
	}
}
