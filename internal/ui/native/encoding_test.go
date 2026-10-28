package native

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// Guard source literals against UTF-8 accidentally decoded as Windows-1252.
func TestUIStringsHaveNoMojibake(t *testing.T) {
	damaged := regexp.MustCompile(`[\x{00c2}\x{00c3}][\x{0080}-\x{00bf}]|\x{00e2}[\x{0080}-\x{00bf}\x{20ac}\x{2013}\x{2014}\x{2018}\x{2019}]|\x{fffd}`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			literal, ok := n.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			if damaged.MatchString(value) {
				t.Errorf("%s: damaged UI string %q", fset.Position(literal.Pos()), value)
			}
			return true
		})
	}
}
