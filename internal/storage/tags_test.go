package storage

import (
	"reflect"
	"strings"
	"testing"
)

func TestInlineTagRules(t *testing.T) {
	tests := []struct {
		name, content string
		want          []string
	}{
		{"backtick fence", "```c\n#include\n#define\n```\n#real", []string{"real"}},
		{"tilde fence", "~~~\n#fake\n~~~\n#real", []string{"real"}},
		{"long fence", "````\n```\n#fake\n~~~~\n`````\n#real", []string{"real"}},
		{"closing info is not a close", "```\n``` info\n#fake\n```\n#real", []string{"real"}},
		{"unclosed fence", "#real\n~~~\n#fake", []string{"real"}},
		{"shorter fence stays open", "````\n```\n#fake\n````\n#real", []string{"real"}},
		{"nested list fence", "- item\n  - nested\n    ~~~\n    #fake\n    ~~~\n#real", []string{"real"}},
		{"list fence", "- item\n    ```\n    #fake\n    ```\n#real", []string{"real"}},
		{"list fence ends with list", "- item\n    ```\n    #fake\n#real", []string{"real"}},
		{"inline code", "`#inline` `` #double ` #fake `` #real", []string{"real"}},
		{"multiline code", "`code\n#fake\n` #real", []string{"real"}},
		{"unmatched code", "` #real", []string{"real"}},
		{"code boundary", "`code`#fake #real", []string{"real"}},
		{"urls", "https://x.com/page#section <https://a/#b> [t](note.md#heading) [t](a(b) #fake) #real", []string{"real"}},
		{"unclosed destination is prose", "[t](broken #real", []string{"real"}},
		{"escaped destination parentheses", "[t](a\\(b\\) #fake) #real", []string{"real"}},
		{"wikilinks", "[[note#heading]] [[ #fake ]] #real", []string{"real"}},
		{"numbers", "#123 #１２３ #1a #y2024 #123/abc", []string{"1a", "y2024"}},
		{"unicode", "#café #日本語 #日本 #a/b/c #under_score #with-dash", []string{"a", "café", "under_score", "with-dash", "日本", "日本語"}},
		{"boundaries", "C# F# foo#bar \\#esc # Heading\n#Title\u2003#Space", []string{"space", "title"}},
		{"frontmatter", "---\r\ntags: '#fake'\r\n---\r\n#real", []string{"real"}},
		{"frontmatter dots", "---\nvalue: #fake\n...\n#real", []string{"real"}},
		{"unclosed frontmatter", "---\n#fake", nil},
		{"later rule", "#real\n---\n#other", []string{"other", "real"}},
		{"kanban", "#kb/todo #KB/done #kb/ #kbx #kb", []string{"kb", "kbx"}},
		{"task tags and duplicates", "- [ ] task #Zebra #apple #APPLE", []string{"apple", "zebra"}},
		{"long line", strings.Repeat("x", 100000) + " #real", []string{"real"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(t.TempDir()).extractTags(tt.content); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("tags = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagConfiguredKanbanPrefix(t *testing.T) {
	previous := KanbanTag
	t.Cleanup(func() { KanbanTag = previous })
	KanbanTag = "Board"
	got := New(t.TempDir()).extractTags("#BOARD/todo #board #kb/todo")
	want := []string{"board", "kb"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %v, want %v", got, want)
	}
}
