package langserver

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/saibing/bingo/langserver/internal/cache"
	"github.com/saibing/bingo/langserver/internal/util"
)

var renameContext = newTestContext(cache.Always)

func TestRenaming(t *testing.T) {
	t.Parallel()

	renameContext.setup(t)

	test := func(t *testing.T, input string, output map[string]string) {
		testRenaming(t, &renamingTestCase{input: input, output: output})
	}

	t.Run("renaming help", func(t *testing.T) {
		test(t, "renaming/a.go:5:2", map[string]string{
			"4:1-4:4":   "renaming/a.go",
			"5:13-5:16": "renaming/a.go",
		})

		test(t, "renaming/a.go:9:6", map[string]string{
			"4:8-4:9": "renaming/a.go",
			"8:5-8:6": "renaming/a.go",
		})

		test(t, "renaming/cgo/a.go:10:2", map[string]string{
			"9:1-9:4":     "renaming/cgo/a.go",
			"10:13-10:16": "renaming/cgo/a.go",
		})

		test(t, "renaming/cgo/a.go:14:6", map[string]string{
			"9:8-9:9":   "renaming/cgo/a.go",
			"13:5-13:6": "renaming/cgo/a.go",
		})
	})
}

type renamingTestCase struct {
	input  string
	output map[string]string
}

func testRenaming(tb testing.TB, c *renamingTestCase) {
	tbRun(tb, fmt.Sprintf("renaming-%s", strings.Replace(c.input, "/", "-", -1)), func(t testing.TB) {
		dir, err := filepath.Abs(renameContext.root())
		if err != nil {
			log.Fatal("testRenaming", err)
		}
		doRenamingTest(t, renameContext.ctx, renameContext.conn, util.PathToURI(dir), c.input, c.output)
	})
}

func doRenamingTest(t testing.TB, ctx context.Context, c *jsonrpc2.Conn, rootURI lsp.DocumentURI, pos string, want map[string]string) {
	file, line, char, err := parsePos(pos)
	if err != nil {
		t.Fatal(err)
	}

	workspaceEdit, err := callRenaming(ctx, c, uriJoin(rootURI, file), line, char, "")
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	for file, edits := range workspaceEdit.Changes {
		for _, edit := range edits {
			got[edit.Range.String()] = filepath.ToSlash(util.UriToRealPath(lsp.DocumentURI(file)))
		}
	}

	for k := range want {
		want[k] = makePath(renameContext.root(), want[k])
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("\ngot %v, \nwant: %v", got, want)
	}
}

func callRenaming(ctx context.Context, c *jsonrpc2.Conn, uri lsp.DocumentURI, line, char int, newName string) (lsp.WorkspaceEdit, error) {
	var edit lsp.WorkspaceEdit
	err := c.Call(ctx, "textDocument/rename", lsp.RenameParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		Position:     lsp.Position{Line: line, Character: char},
		NewName:      newName,
	}, &edit)
	return edit, err
}
