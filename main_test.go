package main

import (
	"go/ast"
	"go/token"
	"reflect"
	"testing"
)

// Handle conversion, see https://stackoverflow.com/questions/12994679/slice-of-struct-slice-of-interface-it-implements
func sliceOfExpressions[T ast.Expr](in []T) []ast.Expr {
	ret := make([]ast.Expr, len(in))
	for i, v := range in {
		ret[i] = ast.Expr(v)
	}
	return ret
}

func TestGetValuesForRange(t *testing.T) {

	correctCond := &ast.BinaryExpr{
		X: &ast.Ident{
			Name: "i",
		},
		Y: &ast.BasicLit{
			Value: "5",
			Kind:  token.INT,
		},
		Op: token.LSS,
	}
	correctInit := &ast.AssignStmt{
		Lhs: sliceOfExpressions([]*ast.Ident{
			{
				Name: "i",
			},
		}),
		Tok: token.DEFINE,
		Rhs: sliceOfExpressions([]*ast.BasicLit{
			{
				Value: "0",
				Kind:  token.INT,
			},
		}),
	}
	correctPost := &ast.IncDecStmt{
		X:   ast.NewIdent("i"),
		Tok: token.INC,
	}
	cases := []struct {
		description string
		stmt        ast.ForStmt
		expVariable string
		expRange    ast.Expr
	}{
		{
			description: "Correct",
			stmt: ast.ForStmt{
				Init: correctInit,
				Cond: correctCond,
				Post: correctPost,
			},
			expVariable: "i",
			expRange: &ast.BasicLit{
				Value: "5",
				Kind:  token.INT,
			},
		},
		{
			description: "While loop",
			stmt:        ast.ForStmt{},
		},
		{
			description: "Init is not assignment",
			stmt: ast.ForStmt{
				Init: &ast.IncDecStmt{},
				Cond: correctCond,
				Post: correctPost,
			},
		},
		{
			description: "More than one variable to be assigned",
			stmt: ast.ForStmt{
				Init: &ast.AssignStmt{
					Lhs: sliceOfExpressions([]*ast.Ident{
						{
							Name: "i",
						},
						{
							Name: "j",
						},
					}),
					Tok: token.DEFINE,
					Rhs: sliceOfExpressions([]*ast.BasicLit{
						{
							Value: "0",
							Kind:  token.INT,
						},
					}),
				},
				Cond: correctCond,
				Post: correctPost,
			},
		},
		{
			description: "Assigning to non-variable",
			stmt: ast.ForStmt{
				Init: &ast.AssignStmt{
					Lhs: sliceOfExpressions([]*ast.ParenExpr{
						{
							X: ast.NewIdent("i"),
						},
					}),
					Tok: token.DEFINE,
					Rhs: sliceOfExpressions([]*ast.BasicLit{
						{
							Value: "0",
							Kind:  token.INT,
						},
					}),
				},
				Cond: correctCond,
				Post: correctPost,
			},
		},
		{
			description: "Not defining a new variable",
			stmt: ast.ForStmt{
				Init: &ast.AssignStmt{
					Lhs: sliceOfExpressions([]*ast.Ident{
						{
							Name: "i",
						},
					}),
					Tok: token.ASSIGN,
					Rhs: sliceOfExpressions([]*ast.BasicLit{
						{
							Value: "0",
							Kind:  token.INT,
						},
					}),
				},
				Cond: correctCond,
				Post: correctPost,
			},
		},
		{
			description: "Too many rhs variables",
			stmt: ast.ForStmt{
				Init: &ast.AssignStmt{
					Lhs: sliceOfExpressions([]*ast.Ident{
						{
							Name: "i",
						},
					}),
					Tok: token.DEFINE,
					Rhs: sliceOfExpressions([]*ast.BasicLit{
						{
							Value: "0",
							Kind:  token.INT,
						},
						{
							Value: "0",
							Kind:  token.INT,
						},
					}),
				},
				Cond: correctCond,
				Post: correctPost,
			},
		},
		{
			description: "Assign to a variable",
			stmt: ast.ForStmt{
				Init: &ast.AssignStmt{
					Lhs: sliceOfExpressions([]*ast.Ident{
						{
							Name: "i",
						},
					}),
					Tok: token.DEFINE,
					Rhs: sliceOfExpressions([]*ast.Ident{
						{
							Name: "j",
						},
					}),
				},
				Cond: correctCond,
				Post: correctPost,
			},
		},
		{
			description: "Assign to non-int",
			stmt: ast.ForStmt{
				Init: &ast.AssignStmt{
					Lhs: sliceOfExpressions([]*ast.Ident{
						{
							Name: "i",
						},
					}),
					Tok: token.DEFINE,
					Rhs: sliceOfExpressions([]*ast.BasicLit{
						{
							Value: "j",
							Kind:  token.STRING,
						},
					}),
				},
				Cond: correctCond,
				Post: correctPost,
			},
		},
		{
			description: "Assign to non-0",
			stmt: ast.ForStmt{
				Init: &ast.AssignStmt{
					Lhs: sliceOfExpressions([]*ast.Ident{
						{
							Name: "i",
						},
					}),
					Tok: token.DEFINE,
					Rhs: sliceOfExpressions([]*ast.BasicLit{
						{
							Value: "1",
							Kind:  token.INT,
						},
					}),
				},
				Cond: correctCond,
				Post: correctPost,
			},
		},
		{
			description: "Non-comparison condition",
			stmt: ast.ForStmt{
				Init: correctInit,
				Cond: &ast.ParenExpr{},
				Post: correctPost,
			},
		},
		{
			description: "Less than or equal",
			stmt: ast.ForStmt{
				Init: correctInit,
				Cond: &ast.BinaryExpr{
					X: &ast.Ident{
						Name: "i",
					},
					Y: &ast.BasicLit{
						Value: "5",
						Kind:  token.INT,
					},
					Op: token.LEQ,
				},
				Post: correctPost,
			},
		},
		{
			description: "Less than non variable",
			stmt: ast.ForStmt{
				Init: correctInit,
				Cond: &ast.BinaryExpr{
					X: &ast.BasicLit{
						Value: "0",
						Kind:  token.INT,
					},
					Y: &ast.BasicLit{
						Value: "5",
						Kind:  token.INT,
					},
					Op: token.LSS,
				},
				Post: correctPost,
			},
		},
		{
			description: "Less than wrong variable",
			stmt: ast.ForStmt{
				Init: correctInit,
				Cond: &ast.BinaryExpr{
					X: &ast.Ident{
						Name: "j",
					},
					Y: &ast.BasicLit{
						Value: "5",
						Kind:  token.INT,
					},
					Op: token.LSS,
				},
				Post: correctPost,
			},
		},
		{
			description: "Post not increment",
			stmt: ast.ForStmt{
				Init: correctInit,
				Cond: correctCond,
				Post: &ast.AssignStmt{},
			},
		},
		{
			description: "Post is decrement",
			stmt: ast.ForStmt{
				Init: correctInit,
				Cond: correctCond,
				Post: &ast.IncDecStmt{
					X:   ast.NewIdent("i"),
					Tok: token.DEC,
				},
			},
		},
		{
			description: "Non variable is incremented",
			stmt: ast.ForStmt{
				Init: correctInit,
				Cond: correctCond,
				Post: &ast.IncDecStmt{
					X: &ast.BasicLit{
						Kind:  token.INT,
						Value: "5",
					},
					Tok: token.INC,
				},
			},
		},
		{
			description: "Incorrect variable incremented",
			stmt: ast.ForStmt{
				Init: correctInit,
				Cond: correctCond,
				Post: &ast.IncDecStmt{
					X:   ast.NewIdent("j"),
					Tok: token.INC,
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			actualVariable, actualRange := getValuesForRange(&tc.stmt)
			if tc.expVariable != actualVariable {
				t.Errorf("Expected variable %v, got %v", tc.expVariable, actualVariable)
			}
			if !reflect.DeepEqual(tc.expRange, actualRange) {
				t.Errorf("Expected range %v, got %v", tc.expRange, actualRange)

			}
		})
	}

}
