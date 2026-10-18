package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serialTests change process-wide state, so they run alone: version.Version, PATH, or the memo suite's testMemo.
var serialTests = map[string]bool{
	"TestAboutHandler_WithoutAuth_ReturnsOnlyProductAndVersion": true,
	"TestInstanceUpgradeGetHandler":                             true,
	"TestInstanceUpgradePostHandler":                            true,
	"TestIntegration_PersonalRunnerPairsT3CodeWithNoHumanInput": true,
	"TestIntegration_PermissionSuitesAnswerTheSameWithAMemo":    true,
}

// TestTests_EveryTopLevelTestRunsInParallel keeps the -race run inside the CI timeout, which serial tests outgrew.
func TestTests_EveryTopLevelTestRunsInParallel(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*_test.go")
	require.NoError(t, err)
	fset := token.NewFileSet()
	var serial []string
	for _, name := range files {
		f, err := parser.ParseFile(fset, name, nil, 0)
		require.NoError(t, err)
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") || fn.Name.Name == "TestMain" {
				continue
			}
			if serialTests[fn.Name.Name] || opensParallel(fn) {
				continue
			}
			serial = append(serial, fn.Name.Name)
		}
	}
	assert.Empty(t, serial, "start these with t.Parallel(), or parallel(t) if the memo suite replays them, or list them in serialTests")
}

func opensParallel(fn *ast.FuncDecl) bool {
	if len(fn.Body.List) == 0 {
		return false
	}
	stmt, ok := fn.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name == "Parallel"
	}
	id, ok := call.Fun.(*ast.Ident)
	return ok && id.Name == "parallel"
}
