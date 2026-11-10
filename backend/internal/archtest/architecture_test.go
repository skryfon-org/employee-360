// Package archtest holds architecture guard tests that run with plain
// `go test` (no build tag) so layering regressions fail `make test`.
package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	modulePath       = "github.com/skryfon-org/employee-360/backend"
	domainServicePkg = modulePath + "/internal/domain/service"
	// emailSendAllowedFile is the only file permitted to call EmailService.Send.
	emailSendAllowedFile = "infrastructure/job/email_worker.go"
)

// internalDir is backend/internal, relative to this package directory.
const internalDir = ".."

// goFiles returns every non-test .go file under root, keyed by slash-separated
// path relative to internalDir, parsed with imports (and full AST).
func parseNonTestFiles(t *testing.T, root string) map[string]*ast.File {
	t.Helper()
	fset := token.NewFileSet()
	out := map[string]*ast.File{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if perr != nil {
			return perr
		}
		rel, rerr := filepath.Rel(internalDir, path)
		if rerr != nil {
			return rerr
		}
		out[filepath.ToSlash(rel)] = f
		return nil
	})
	require.NoError(t, err)
	return out
}

// TestUsecaseLayerImportsNoInfrastructure (AC-2): usecases must not import
// River, database/sql, GORM, or any infrastructure package.
func TestUsecaseLayerImportsNoInfrastructure(t *testing.T) {
	files := parseNonTestFiles(t, filepath.Join(internalDir, "usecase"))
	require.NotEmpty(t, files, "no usecase files found; guard would be vacuous")

	forbidden := func(p string) bool {
		return strings.HasPrefix(p, "github.com/riverqueue/") ||
			p == "database/sql" || strings.HasPrefix(p, "database/sql/") ||
			strings.HasPrefix(p, "gorm.io/") ||
			strings.HasPrefix(p, modulePath+"/internal/infrastructure")
	}

	var violations []string
	for name, f := range files {
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			require.NoError(t, err)
			if forbidden(p) {
				violations = append(violations, name+" imports "+p)
			}
		}
	}
	sort.Strings(violations)
	require.Empty(t, violations, "usecase layer must not depend on infrastructure/persistence/queue packages")
}

// TestRedisClientOnlyImportedByInfrastructure: the go-redis client may be
// imported only under internal/infrastructure/ (the Cache port lives in the
// domain, with no Redis types), so Redis never leaks into other layers.
func TestRedisClientOnlyImportedByInfrastructure(t *testing.T) {
	files := parseNonTestFiles(t, internalDir)
	require.NotEmpty(t, files, "no files found; guard would be vacuous")

	var violations []string
	found := false
	for name, f := range files {
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			require.NoError(t, err)
			if !strings.HasPrefix(p, "github.com/redis/") {
				continue
			}
			if strings.HasPrefix(name, "infrastructure/") {
				found = true
				continue
			}
			violations = append(violations, name+" imports "+p)
		}
	}
	sort.Strings(violations)
	require.Empty(t, violations, "go-redis may only be imported under internal/infrastructure/")
	require.True(t, found, "expected the Redis adapter under internal/infrastructure/ to import go-redis")
}

// TestEmailServiceSendOnlyCalledFromWorker (AC-3): the only non-test file that
// calls Send on a domain service.EmailService is job/email_worker.go.
//
// Heuristic (no type checker): per package directory, collect identifiers
// (struct fields, params, vars, consts) whose declared type is EmailService
// (qualified via an import of the domain service package, or unqualified inside
// that package). A call `<expr>.Send(...)` is a violation when the last name in
// <expr> is one of those identifiers. Send methods on unrelated types (queues,
// SMTP clients, ...) are not flagged.
func TestEmailServiceSendOnlyCalledFromWorker(t *testing.T) {
	files := parseNonTestFiles(t, internalDir)

	byDir := map[string][]string{}
	for name := range files {
		dir := filepath.ToSlash(filepath.Dir(name))
		byDir[dir] = append(byDir[dir], name)
	}

	var callers []string
	for dir, names := range byDir {
		emailNames := map[string]bool{}
		for _, name := range names {
			collectEmailServiceNames(files[name], dir == "domain/service", emailNames)
		}
		if len(emailNames) == 0 {
			continue
		}
		for _, name := range names {
			if callsSendOn(files[name], emailNames) {
				callers = append(callers, name)
			}
		}
	}
	sort.Strings(callers)
	require.Equal(t, []string{emailSendAllowedFile}, callers,
		"EmailService.Send must be called only from %s", emailSendAllowedFile)
}

func serviceAliases(f *ast.File) map[string]bool {
	aliases := map[string]bool{}
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		if p != domainServicePkg {
			continue
		}
		if imp.Name != nil {
			aliases[imp.Name.Name] = true
		} else {
			aliases["service"] = true
		}
	}
	return aliases
}

func isEmailServiceType(expr ast.Expr, aliases map[string]bool, inServicePkg bool) bool {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return isEmailServiceType(e.X, aliases, inServicePkg)
	case *ast.SelectorExpr:
		id, ok := e.X.(*ast.Ident)
		return ok && aliases[id.Name] && e.Sel.Name == "EmailService"
	case *ast.Ident:
		return inServicePkg && e.Name == "EmailService"
	}
	return false
}

func collectEmailServiceNames(f *ast.File, inServicePkg bool, out map[string]bool) {
	aliases := serviceAliases(f)
	if len(aliases) == 0 && !inServicePkg {
		return
	}
	addFields := func(fl *ast.FieldList) {
		if fl == nil {
			return
		}
		for _, fld := range fl.List {
			if isEmailServiceType(fld.Type, aliases, inServicePkg) {
				for _, n := range fld.Names {
					out[n.Name] = true
				}
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.StructType:
			addFields(x.Fields)
		case *ast.FuncType:
			addFields(x.Params)
			addFields(x.Results)
		case *ast.ValueSpec:
			if x.Type != nil && isEmailServiceType(x.Type, aliases, inServicePkg) {
				for _, n := range x.Names {
					out[n.Name] = true
				}
			}
		}
		return true
	})
}

func callsSendOn(f *ast.File, emailNames map[string]bool) bool {
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Send" {
			return true
		}
		switch recv := sel.X.(type) {
		case *ast.Ident:
			found = found || emailNames[recv.Name]
		case *ast.SelectorExpr:
			found = found || emailNames[recv.Sel.Name]
		}
		return true
	})
	return found
}
