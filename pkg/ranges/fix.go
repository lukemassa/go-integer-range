package ranges

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path"

	"github.com/tsuna/gorewrite"
)

// If possible, extract the variable name and range expression
func getValuesForRange(stmt *ast.ForStmt) (string, ast.Expr) {
	variable := ""

	// Step 1) Is the init correct?
	assignStmt, ok := stmt.Init.(*ast.AssignStmt)
	//fmt.Println(assignStmt.Tok)
	if !ok {
		return "", nil
	}

	// LHS of statement must be a single variable

	if len(assignStmt.Lhs) != 1 {
		return "", nil
	}

	variableStmt, ok := assignStmt.Lhs[0].(*ast.Ident)
	if !ok {
		return "", nil
	}
	variable = variableStmt.Name

	// RHS of statement must be ":= 0"

	if assignStmt.Tok != token.DEFINE {
		return "", nil
	}
	if len(assignStmt.Rhs) != 1 {
		return "", nil
	}
	assignToStmt, ok := assignStmt.Rhs[0].(*ast.BasicLit)
	if !ok {
		return "", nil
	}
	if assignToStmt.Kind != token.INT {
		return "", nil
	}
	if assignToStmt.Value != "0" {
		return "", nil
	}

	// Step 2) Is the condition correct?

	binaryExpr, ok := stmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return "", nil
	}
	if binaryExpr.Op != token.LSS {
		return "", nil
	}
	lessThanExpr, ok := binaryExpr.X.(*ast.Ident)
	if !ok {
		return "", nil
	}
	if lessThanExpr.Name != variable {
		return "", nil
	}
	// Note: we allow any arbitrary expression here, with the assumption
	// that the incoming code is valid go, hence "i < XXX" where we already
	// asserted that i is an integer, means that XXX will be an integer.
	// For example this could be a function call like len(), which would be
	// hard to detect the type of at the AST stage
	rangeValue := binaryExpr.Y

	// Step 3) Is post correct?
	incExp, ok := stmt.Post.(*ast.IncDecStmt)
	if !ok {
		return "", nil
	}
	if incExp.Tok != token.INC {
		return "", nil
	}
	variableIncremented, ok := incExp.X.(*ast.Ident)
	if !ok {
		return "", nil
	}
	if variableIncremented.Name != variable {
		return "", nil
	}

	return variable, rangeValue
}

// Fix takes a reader that represents a go source file, and returns a reader
// with integer for loops changed to range statements. If no for loops are
// able to converted, returns a nil reader and nil error.
func Fix(in io.Reader) (io.Reader, error) {
	return fix("<file>", in)
}

func fix(filename string, in io.Reader) (io.Reader, error) {
	if in == nil {
		return nil, errors.New("nil reader")
	}
	fset := token.NewFileSet()
	// This won't actually parse the file, since in is non-nil
	node, err := parser.ParseFile(fset, filename, in, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	replacer := forLoopWithIntReplacer{}
	gorewrite.Rewrite(&replacer, node)

	// No updates, no need to write file
	if !replacer.updated {
		return nil, nil

	}
	var b bytes.Buffer

	err = format.Node(&b, fset, node)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// Fix integer loops into ranges for a given file, skipping actually editing the file
// if dryrun is set.
func FixFile(filename string, dryrun bool) error {
	_, err := os.Stat(filename)
	if err != nil {
		return err
	}
	dir := path.Dir(filename)
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	res, err := fix(filename, file)
	if err != nil {
		return err
	}
	// No updates, no need to write file
	if res == nil {
		log.Printf("No updates needed for %s", filename)
		return nil
	}
	if dryrun {
		log.Printf("Would have updated %s, skipping for dryrun", filename)
		return nil
	}

	log.Printf("Updating %s", filename)

	outputFile, err := os.CreateTemp(dir, fmt.Sprintf("%s.new.", path.Base(filename)))
	if err != nil {
		log.Fatal(err)
	}
	outputFileName := outputFile.Name()

	defer func() {
		outputFile.Close()
		_, err := os.Stat(outputFileName)
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		if err != nil {
			panic(err)
		}
		err = os.Remove(outputFileName)
		if err != nil {
			panic(err)
		}
	}()
	io.Copy(outputFile, res)

	err = os.Rename(outputFileName, filename)
	if err != nil {
		return err
	}
	return nil
}

type forLoopWithIntReplacer struct {
	updated bool
}

// Rewrite implements gorewrite.Rewriter to take a given node in an AST,
// and, if it's a for loop that can be converted into a range staement, do so
func (v *forLoopWithIntReplacer) Rewrite(n ast.Node) (ast.Node, gorewrite.Rewriter) {
	forStmt, ok := n.(*ast.ForStmt)
	if !ok {
		return n, v
	}
	variable, rangeValue := getValuesForRange(forStmt)
	// Something about the for loop was not conducive to being a range, simply return as is
	if variable == "" || rangeValue == nil {
		return n, v
	}
	v.updated = true
	return &ast.RangeStmt{
		Key: &ast.Ident{
			Name: variable,
		},
		Tok:  token.DEFINE,
		X:    rangeValue,
		Body: forStmt.Body,
	}, v

}
