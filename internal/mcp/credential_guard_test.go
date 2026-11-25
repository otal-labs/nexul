package mcp

import (
	"go/ast"
	"go/constant"
	"go/types"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

// credentialField names a result field that can carry a token, a secret, or a link, code or command with one inside.
var credentialField = regexp.MustCompile(`(^|_)(token|secret|password|passphrase|credential|url|link|command|code)s?($|_)|(api|private|secret|access|signing)_key`)

// notCredentials lists the fields credentialField matches that hold no credential, with why: a bare field name
// covers it in every tool, "tool path" covers one tool's field.
var notCredentials = map[string]string{
	"avatar_url":         "a profile picture's address",
	"account_avatar_url": "a profile picture's address",
	"html_url":           "a GitHub page's address",
	"issuer_url":         "the SSO provider's public issuer",
	"origin_url":         "the local service a tunnel forwards to",
	"instance_url":       "this instance's own address",
	"server_url":         "a computer's harness address, which takes a token of its own",
	"secrets_changed":    "the names of variables whose secret changed, never a value",
	"stream_url":         "where the audit stream posts, refused with a user or password in it; its signing secret is never returned",

	"attachment_create url":                "the attachment's path, which needs the reader's own sign-in",
	"ticket_get test_target.url":           "a preview or test environment's address",
	"topology_get nodes.data.url":          "an external node's address",
	"topology_update nodes.data.url":       "an external node's address",
	"instance_update saml.metadata_url":    "the SAML provider's public metadata",
	"instance_update saml.idp_sso_url":     "the SAML provider's public sign-in page",
	"instance_update saml.sp.acs_url":      "this instance's own public SAML endpoint",
	"instance_update saml.sp.metadata_url": "this instance's own public SAML metadata",
	"instance_get latest.url":              "the latest release's page",
	"instance_get logs.items.url":          "the page a browser log line came from, with tokens redacted",
	"meeting_list items.link":              "where a meeting takes place, as a person typed it",
	"meeting_create link":                  "where a meeting takes place, as a person typed it",
	"meeting_update link":                  "where a meeting takes place, as a person typed it",
	"computer_list items.mcp_token":        "the token's id, name and prefix, never the token",
	"computer_pair mcp_token":              "the token's id, name and prefix, never the token",
	"stack_create stack.command":           "the container command the caller wrote, echoed back",
	"computer_list items.commands":         "the commands run on the computer and how each ended",
	"computer_list items.commands.command": "a command line its owner or their agent ran, echoed back",
	"computer_pair commands":               "never set by this tool; computer_list fills it",
	"machine_list commands":                "the commands run on the machine's runners and how each ended",
	"machine_list commands.command":        "a command line someone holding Run commands on the machine ran, echoed back",
	"machine_list items.commands":          "never set on the list; the machine read alone with id carries them",
	"machine_list items.commands.command":  "never set on the list; the machine read alone with id carries them",
	"computer_pair commands.command":       "never set by this tool; computer_list fills it",
}

// A sign-in grant is refused only the tools marked MintsCredential (ADR 0204), so a tool that returns a credential
// without the mark hands a grant something that outlives its revoke.
func TestToolResults_CredentialsComeOnlyFromToolsMarkedMintsCredential(t *testing.T) {
	results := toolResultFields(t)
	used := map[string]bool{}
	for _, tool := range surface() {
		fields, ok := results[tool.Name]
		require.True(t, ok, "%s: no mcptool.New call declares it, so its result could not be read", tool.Name)
		var credentials []string
		for _, f := range fields {
			leaf := f[strings.LastIndex(f, ".")+1:]
			if !credentialField.MatchString(leaf) {
				continue
			}
			if _, ok := notCredentials[leaf]; ok {
				used[leaf] = true
				continue
			}
			if _, ok := notCredentials[tool.Name+" "+f]; ok {
				used[tool.Name+" "+f] = true
				continue
			}
			credentials = append(credentials, f)
		}
		if tool.Hints.MintsCredential {
			assert.NotEmpty(t, credentials, "%s is marked MintsCredential but returns no field credentialField recognizes; name the credential so the guard sees it", tool.Name)
			continue
		}
		for _, f := range credentials {
			assert.Failf(t, "unmarked credential", "%s returns %s, which reads as a credential: mark the tool mcptool.Hints.MintsCredential, or list the field in notCredentials with why it holds none", tool.Name, f)
		}
	}
	for key := range notCredentials {
		assert.True(t, used[key], "notCredentials lists %q, which no unmarked tool returns any more; drop it", key)
	}
}

// toolResultFields reads every mcptool.New call in the module and returns, per tool name, the json field paths of
// every type its handler can return.
func toolResultFields(t *testing.T) map[string][]string {
	t.Helper()
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo, Dir: "../.."}
	pkgs, err := packages.Load(cfg, "./...")
	require.NoError(t, err)
	require.Zero(t, packages.PrintErrors(pkgs), "the module must type-check")

	r := newReturnReader(pkgs)
	out := map[string][]string{}
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || !isMCPToolNew(p.TypesInfo, call) {
					return true
				}
				at := p.Fset.Position(call.Pos())
				name := p.TypesInfo.Types[call.Args[0]].Value
				require.NotNil(t, name, "%s: a tool's name is a constant", at)
				tool := constant.StringVal(name)
				results := r.handlerResults(p.TypesInfo, call.Args[4])
				assert.NotEmpty(t, results, "%s (%s): no value its handler returns could be found", tool, at)
				fields := []string{}
				for _, res := range results {
					if res.keys != nil {
						for key, typ := range res.keys {
							if !holdsNoText(typ) {
								fields = append(fields, key)
							}
							fields = append(fields, jsonFields(typ, key+".", map[*types.Named]bool{})...)
						}
						continue
					}
					if opaque(res.typ) {
						assert.Failf(t, "result not readable", "%s (%s) returns a value typed %s; return a concrete type, or a map literal with constant keys", tool, at, res.typ)
						continue
					}
					fields = append(fields, jsonFields(res.typ, "", map[*types.Named]bool{})...)
				}
				slices.Sort(fields)
				out[tool] = slices.Compact(fields)
				return true
			})
		}
	}
	return out
}

func isMCPToolNew(info *types.Info, call *ast.CallExpr) bool {
	fun := call.Fun
	if ix, ok := fun.(*ast.IndexExpr); ok {
		fun = ix.X
	}
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	fn, ok := info.Uses[sel.Sel].(*types.Func)
	return ok && fn.Name() == "New" && fn.Pkg() != nil && fn.Pkg().Path() == "github.com/otal-labs/nexul/internal/platform/mcptool"
}

// result is one value a handler can return: its static type, and for a map literal with constant keys, each key's
// value type.
type result struct {
	typ  types.Type
	keys map[string]types.Type
}

// opaque is an interface or a map built other than as a constant-key literal, whose json keys no type says.
func opaque(typ types.Type) bool {
	_, isMap := typ.Underlying().(*types.Map)
	return isMap || types.IsInterface(typ)
}

type funcBody struct {
	body *ast.BlockStmt
	info *types.Info
}

// returnReader collects the values a handler's first result can take, following any call that returns an
// interface into the function, or every implementation of the interface method, it reaches.
type returnReader struct {
	bodies map[*types.Func]funcBody
	named  []*types.Named
}

func newReturnReader(pkgs []*packages.Package) returnReader {
	r := returnReader{bodies: map[*types.Func]funcBody{}}
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			for _, d := range f.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok || fd.Body == nil {
					continue
				}
				if fn, ok := p.TypesInfo.Defs[fd.Name].(*types.Func); ok {
					r.bodies[fn] = funcBody{fd.Body, p.TypesInfo}
				}
			}
		}
		scope := p.Types.Scope()
		for _, name := range scope.Names() {
			tn, ok := scope.Lookup(name).(*types.TypeName)
			if !ok {
				continue
			}
			if n, ok := tn.Type().(*types.Named); ok && n.TypeParams() == nil && !types.IsInterface(n) {
				r.named = append(r.named, n)
			}
		}
	}
	return r
}

func (r returnReader) handlerResults(info *types.Info, handler ast.Expr) []result {
	if lit, ok := handler.(*ast.FuncLit); ok {
		return r.returns(info, lit.Body, map[*types.Func]bool{})
	}
	return r.calleeResults(info, handler, map[*types.Func]bool{})
}

// calleeResults reads the function fun names, or every implementation when it is an interface method; nil when
// there is no body to read.
func (r returnReader) calleeResults(info *types.Info, fun ast.Expr, seen map[*types.Func]bool) []result {
	var id *ast.Ident
	switch e := ast.Unparen(fun).(type) {
	case *ast.Ident:
		id = e
	case *ast.SelectorExpr:
		id = e.Sel
	}
	if id == nil {
		return nil
	}
	fn, ok := info.Uses[id].(*types.Func)
	if !ok {
		return nil
	}
	var out []result
	for _, impl := range r.implementations(fn.Origin()) {
		b, ok := r.bodies[impl]
		if !ok || seen[impl] {
			continue
		}
		seen[impl] = true
		out = append(out, r.returns(b.info, b.body, seen)...)
	}
	return out
}

func (r returnReader) implementations(fn *types.Func) []*types.Func {
	recv := fn.Signature().Recv()
	if recv == nil || !types.IsInterface(recv.Type()) {
		return []*types.Func{fn}
	}
	iface := recv.Type().Underlying().(*types.Interface)
	var out []*types.Func
	for _, n := range r.named {
		for _, typ := range []types.Type{n, types.NewPointer(n)} {
			if !types.Implements(typ, iface) {
				continue
			}
			obj, _, _ := types.LookupFieldOrMethod(typ, false, fn.Pkg(), fn.Name())
			if m, ok := obj.(*types.Func); ok {
				out = append(out, m)
			}
			break
		}
	}
	return out
}

func (r returnReader) returns(info *types.Info, body *ast.BlockStmt, seen map[*types.Func]bool) []result {
	var out []result
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			return true
		}
		e := ret.Results[0]
		typ := info.TypeOf(e)
		if tuple, ok := typ.(*types.Tuple); ok {
			typ = tuple.At(0).Type()
		}
		if b, ok := typ.(*types.Basic); ok && b.Kind() == types.UntypedNil {
			return true
		}
		if call, ok := ast.Unparen(e).(*ast.CallExpr); ok && types.IsInterface(typ) {
			if more := r.calleeResults(info, call.Fun, seen); more != nil {
				out = append(out, more...)
				return true
			}
		}
		out = append(out, result{typ: typ, keys: literalKeys(info, e)})
		return true
	})
	return out
}

// literalKeys reads a map literal's keys and their value types; nil when e is no map literal or a key is not constant.
func literalKeys(info *types.Info, e ast.Expr) map[string]types.Type {
	lit, ok := ast.Unparen(e).(*ast.CompositeLit)
	if !ok {
		return nil
	}
	if _, ok := info.TypeOf(lit).Underlying().(*types.Map); !ok {
		return nil
	}
	keys := map[string]types.Type{}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return nil
		}
		key := info.Types[kv.Key].Value
		if key == nil || key.Kind() != constant.String {
			return nil
		}
		keys[constant.StringVal(key)] = info.TypeOf(kv.Value)
	}
	return keys
}

// jsonFields lists the json paths a value of typ encodes to that can hold text, joined by dots; pointers, slices and
// maps are seen through.
func jsonFields(typ types.Type, prefix string, seen map[*types.Named]bool) []string {
	switch t := types.Unalias(typ).(type) {
	case *types.Pointer:
		return jsonFields(t.Elem(), prefix, seen)
	case *types.Slice:
		return jsonFields(t.Elem(), prefix, seen)
	case *types.Array:
		return jsonFields(t.Elem(), prefix, seen)
	case *types.Map:
		return jsonFields(t.Elem(), prefix, seen)
	case *types.Named:
		if seen[t] {
			return nil
		}
		seen[t] = true
		defer delete(seen, t)
		return jsonFields(t.Underlying(), prefix, seen)
	case *types.Struct:
		var out []string
		for i := range t.NumFields() {
			f := t.Field(i)
			if !f.Exported() {
				continue
			}
			name, _, _ := strings.Cut(reflect.StructTag(t.Tag(i)).Get("json"), ",")
			if name == "-" || holdsNoText(f.Type()) {
				continue
			}
			if name == "" && f.Embedded() {
				out = append(out, jsonFields(f.Type(), prefix, seen)...)
				continue
			}
			if name == "" {
				name = f.Name()
			}
			out = append(out, prefix+name)
			out = append(out, jsonFields(f.Type(), prefix+name+".", seen)...)
		}
		return out
	}
	return nil
}

// holdsNoText is a bool or a number, such as token_revoked or exit_code, which no credential fits in.
func holdsNoText(typ types.Type) bool {
	if p, ok := typ.Underlying().(*types.Pointer); ok {
		typ = p.Elem()
	}
	b, ok := typ.Underlying().(*types.Basic)
	return ok && b.Info()&types.IsString == 0
}
