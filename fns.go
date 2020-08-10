package main

import (
	"strings"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
)



type fnTag struct {
	tag      string
	node     *ast.Ident
	pkg      string
	file     string
	fn       string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage fns <package>")
		os.Exit(1)
	}

	pkg := os.Args[1]

	set := token.NewFileSet()
	packs, err := parser.ParseDir(set, pkg, nil, 0)
	if err != nil {
		 fmt.Println("Failed to parse package:", err)
		 os.Exit(1)
	}

	funcs := []fnTag{}
	for _, pack := range packs {
		 for file, f := range pack.Files {
			  for _, d := range f.Decls {
					if fn, isFn := d.(*ast.FuncDecl); isFn {
						node, tag, ok := getFnTag( fn )
						if ok {
							funcs = append( funcs, fnTag{ tag:tag, node: node, pkg: pack.Name, file: file, fn: fn.Name.Name } )
						}

					}
			  }
		 }
	}

	for _, fn := range funcs {
		fileName := filepath.Base( fn.file )
		extension := filepath.Ext(fileName)
		fileName = fileName[0:len(fileName)-len(extension)]
		correct := fmt.Sprintf("%s.%s.%s", fn.pkg, fileName, fn.fn )
		if fn.tag != correct {
			fmt.Printf("%s:%d Incorrect fn tag %q. Should be %q\n", fn.file, set.Position( fn.node.Pos() ).Line, fn.tag, correct )
		}
	}
}



func getFnTag(fn *ast.FuncDecl) (*ast.Ident, string, bool) {
	block := fn.Body
	for _, stmt := range block.List {
		ass, ok := stmt.(*ast.AssignStmt )
		if !ok {
			continue
		}

		if len( ass.Lhs ) != 1 || len( ass.Rhs ) != 1 {
			continue
		}

		variable, ok := ass.Lhs[ 0 ].(*ast.Ident)
		if !ok {
			continue
		}

		if variable.Name == "fn" {
			val, ok := ass.Rhs[ 0 ].(*ast.BasicLit)
			if !ok {
				continue
			}

			return variable, strings.Trim( val.Value, `"` ), true
		}
	}
	return nil, "", false
}
