// Package hash provides structural hashing of Go AST nodes.
// Two nodes with identical structure but different identifier names
// or literal values produce the same hash - enabling detection of
// copy-pasted code that has been superficially modified.
package hash

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/token"

	"godedup/internal/wrapx"
)

// FuncInfo holds the hashing result for a single function.
type FuncInfo struct {
	Name     string   // fully qualified: "pkg.Func" or "pkg.(Type).Method"
	File     string   // source file path
	Line     int      // line number of declaration
	TopHash  uint64   // hash of the entire function body
	StmtSeq  []uint64 // per-statement hashes for similarity comparison
	NumStmts int      // total statement count (excluding blank lines)
	NumLines int      // line span of the function body
}

// Hasher computes structural hashes of AST nodes.
type Hasher struct {
	fset *token.FileSet
}

func New(fset *token.FileSet) *Hasher {
	return &Hasher{fset: fset}
}

// HashFunc computes the FuncInfo for a function declaration.
func (h *Hasher) HashFunc(pkg, file string, fn *ast.FuncDecl) FuncInfo {
	if fn.Body == nil {
		return FuncInfo{}
	}

	name := qualifiedName(pkg, fn)
	pos := h.fset.Position(fn.Pos())
	endPos := h.fset.Position(fn.End())

	// per-statement hashes
	stmtSeq := make([]uint64, 0, len(fn.Body.List))
	for _, stmt := range fn.Body.List {
		stmtSeq = append(stmtSeq, h.hashNode(stmt))
	}

	// top-level hash: hash of the statement sequence
	topHash := hashUint64Slice(stmtSeq)

	return FuncInfo{
		Name:     name,
		File:     file,
		Line:     pos.Line,
		TopHash:  topHash,
		StmtSeq:  stmtSeq,
		NumStmts: len(stmtSeq),
		NumLines: endPos.Line - pos.Line + 1,
	}
}

// hashNode recursively hashes an AST node, normalizing away
// identifier names and literal values.
//
//nolint:gocyclo
func (h *Hasher) hashNode(node ast.Node) uint64 {
	if node == nil {
		return 0
	}

	switch n := node.(type) {
	// Statements

	case *ast.BlockStmt:
		children := make([]uint64, 0, len(n.List))
		for _, s := range n.List {
			children = append(children, h.hashNode(s))
		}
		return combine(nodeID("BlockStmt"), hashUint64Slice(children))

	case *ast.IfStmt:
		return combine(nodeID("IfStmt"),
			h.hashNode(n.Init),
			h.hashNode(n.Cond),
			h.hashNode(n.Body),
			h.hashNode(n.Else),
		)

	case *ast.ForStmt:
		return combine(nodeID("ForStmt"),
			h.hashNode(n.Init),
			h.hashNode(n.Cond),
			h.hashNode(n.Post),
			h.hashNode(n.Body),
		)

	case *ast.RangeStmt:
		return combine(nodeID("RangeStmt"),
			h.hashNode(n.X), // what we range over (normalized)
			h.hashNode(n.Body),
		)

	case *ast.SwitchStmt:
		return combine(nodeID("SwitchStmt"),
			h.hashNode(n.Init),
			h.hashNode(n.Tag),
			h.hashNode(n.Body),
		)

	case *ast.TypeSwitchStmt:
		return combine(nodeID("TypeSwitchStmt"),
			h.hashNode(n.Init),
			h.hashNode(n.Assign),
			h.hashNode(n.Body),
		)

	case *ast.SelectStmt:
		return combine(nodeID("SelectStmt"), h.hashNode(n.Body))

	case *ast.CommClause:
		return combine(nodeID("CommClause"),
			h.hashNode(n.Comm),
			h.hashStmtList(n.Body),
		)

	case *ast.CaseClause:
		return combine(nodeID("CaseClause"),
			h.hashExprList(n.List),
			h.hashStmtList(n.Body),
		)

	case *ast.ReturnStmt:
		return combine(nodeID("ReturnStmt"), h.hashExprList(n.Results))

	case *ast.AssignStmt:
		return combine(nodeID("AssignStmt"),
			wrapx.ToUint64(n.Tok), // = vs :=
			h.hashExprList(n.Lhs),
			h.hashExprList(n.Rhs),
		)

	case *ast.ExprStmt:
		return combine(nodeID("ExprStmt"), h.hashNode(n.X))

	case *ast.DeclStmt:
		return combine(nodeID("DeclStmt"), h.hashNode(n.Decl))

	case *ast.GenDecl:
		specs := make([]uint64, 0, len(n.Specs))
		for _, s := range n.Specs {
			specs = append(specs, h.hashNode(s))
		}
		return combine(nodeID("GenDecl"), wrapx.ToUint64(n.Tok), hashUint64Slice(specs))

	case *ast.ValueSpec:
		return combine(nodeID("ValueSpec"),
			h.hashExprList(n.Values),
			h.hashNode(n.Type),
		)

	case *ast.IncDecStmt:
		return combine(nodeID("IncDecStmt"), wrapx.ToUint64(n.Tok), h.hashNode(n.X))

	case *ast.SendStmt:
		return combine(nodeID("SendStmt"),
			h.hashNode(n.Chan),
			h.hashNode(n.Value),
		)

	case *ast.GoStmt:
		return combine(nodeID("GoStmt"), h.hashNode(n.Call))

	case *ast.DeferStmt:
		return combine(nodeID("DeferStmt"), h.hashNode(n.Call))

	case *ast.BranchStmt:
		return combine(nodeID("BranchStmt"), wrapx.ToUint64(n.Tok))

	case *ast.LabeledStmt:
		// normalize away label name
		return combine(nodeID("LabeledStmt"), h.hashNode(n.Stmt))

	// Expressions

	case *ast.Ident:
		// normalize: all identifiers hash the same
		// EXCEPT nil, true, false - these are semantic
		switch n.Name {
		case "nil":
			return nodeID("nil")
		case "true":
			return nodeID("true")
		case "false":
			return nodeID("false")
		}
		return nodeID("Ident") // all other identifiers are equivalent

	case *ast.BasicLit:
		// normalize: all literals of the same kind hash the same
		return combine(nodeID("BasicLit"), wrapx.ToUint64(n.Kind))

	case *ast.BinaryExpr:
		return combine(nodeID("BinaryExpr"),
			wrapx.ToUint64(n.Op),
			h.hashNode(n.X),
			h.hashNode(n.Y),
		)

	case *ast.UnaryExpr:
		return combine(nodeID("UnaryExpr"), wrapx.ToUint64(n.Op), h.hashNode(n.X))

	case *ast.CallExpr:
		return combine(nodeID("CallExpr"),
			h.hashNode(n.Fun),
			h.hashExprList(n.Args),
		)

	case *ast.SelectorExpr:
		// normalize package/receiver: only the selector name matters
		// fmt.Println and log.Println -> same structure
		return combine(nodeID("SelectorExpr"), nodeID(n.Sel.Name))

	case *ast.IndexExpr:
		return combine(nodeID("IndexExpr"),
			h.hashNode(n.X),
			h.hashNode(n.Index),
		)

	case *ast.SliceExpr:
		return combine(nodeID("SliceExpr"),
			h.hashNode(n.X),
			h.hashNode(n.Low),
			h.hashNode(n.High),
		)

	case *ast.StarExpr:
		return combine(nodeID("StarExpr"), h.hashNode(n.X))

	case *ast.CompositeLit:
		return combine(nodeID("CompositeLit"),
			h.hashNode(n.Type),
			h.hashExprList(n.Elts),
		)

	case *ast.KeyValueExpr:
		return combine(nodeID("KeyValueExpr"),
			h.hashNode(n.Key),
			h.hashNode(n.Value),
		)

	case *ast.TypeAssertExpr:
		return combine(nodeID("TypeAssertExpr"),
			h.hashNode(n.X),
			h.hashNode(n.Type),
		)

	case *ast.FuncLit:
		return combine(nodeID("FuncLit"), h.hashNode(n.Body))

	case *ast.ParenExpr:
		return h.hashNode(n.X) // parens are transparent

	// Types

	case *ast.ArrayType:
		return combine(nodeID("ArrayType"), h.hashNode(n.Elt))

	case *ast.MapType:
		return combine(nodeID("MapType"),
			h.hashNode(n.Key),
			h.hashNode(n.Value),
		)

	case *ast.ChanType:
		return combine(nodeID("ChanType"), wrapx.ToUint64(n.Dir), h.hashNode(n.Value))

	case *ast.InterfaceType:
		return nodeID("InterfaceType")

	case *ast.StructType:
		return nodeID("StructType")
	}

	return 0
}

// hashNodeList hashes a slice of AST nodes into a single uint64.
// ast.Expr and ast.Stmt both implement ast.Node so this covers both cases.
func (h *Hasher) hashNodeList(nodes []ast.Node) uint64 {
	hashes := make([]uint64, 0, len(nodes))
	for _, n := range nodes {
		hashes = append(hashes, h.hashNode(n))
	}
	return hashUint64Slice(hashes)
}

func (h *Hasher) hashExprList(exprs []ast.Expr) uint64 {
	nodes := make([]ast.Node, len(exprs))
	for i, e := range exprs {
		nodes[i] = e
	}
	return h.hashNodeList(nodes)
}

func (h *Hasher) hashStmtList(stmts []ast.Stmt) uint64 {
	nodes := make([]ast.Node, len(stmts))
	for i, s := range stmts {
		nodes[i] = s
	}
	return h.hashNodeList(nodes)
}

// Helpers

// nodeID maps a node type name to a stable uint64.
func nodeID(name string) uint64 {
	h := sha256.Sum256([]byte(name))
	return binary.LittleEndian.Uint64(h[:8])
}

// combine mixes multiple uint64 values using FNV-style mixing.
func combine(vals ...uint64) uint64 {
	var h uint64 = 0xcbf29ce484222325 // FNV offset basis
	for _, v := range vals {
		h ^= v
		h *= 0x100000001b3 // FNV prime
	}
	return h
}

// hashUint64Slice hashes a sequence of uint64s order-dependently.
func hashUint64Slice(s []uint64) uint64 {
	if len(s) == 0 {
		return 0
	}
	var h uint64 = 0xcbf29ce484222325
	for _, v := range s {
		h ^= v
		h *= 0x100000001b3
		// mix position-dependence in
		h ^= h >> 17
	}
	return h
}

// qualifiedName returns "pkg.FuncName" or "pkg.(ReceiverType).MethodName".
func qualifiedName(pkg string, fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fmt.Sprintf("%s.%s", pkg, fn.Name.Name)
	}
	recv := fn.Recv.List[0].Type
	recvStr := "?"
	switch r := recv.(type) {
	case *ast.Ident:
		recvStr = r.Name
	case *ast.StarExpr:
		if id, ok := r.X.(*ast.Ident); ok {
			recvStr = "*" + id.Name
		}
	}
	return fmt.Sprintf("%s.(%s).%s", pkg, recvStr, fn.Name.Name)
}
