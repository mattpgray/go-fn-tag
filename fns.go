package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type fnTag struct {
	tag  string
	node *ast.Ident
	pkg  string
	file string
	fn   string
}

func (t *fnTag) expected() string {
	fileName := filepath.Base(t.file)
	extension := filepath.Ext(fileName)
	fileName = fileName[0 : len(fileName)-len(extension)]
	return fmt.Sprintf("%s.%s.%s", t.pkg, fileName, t.fn)
}

func main() {
	write := flag.Bool("w", false, "Rewrite the files with correct fn tags")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Println("Usage fns <package>")
		os.Exit(1)
	}

	pkg := flag.Args()[0]

	set := token.NewFileSet()
	packs, err := parser.ParseDir(set, pkg, nil, parser.ParseComments)
	if err != nil {
		fmt.Println("Failed to parse package:", err)
		os.Exit(1)
	}

	funcs := []fnTag{}
	for _, pack := range packs {
		for file, f := range pack.Files {
			for _, d := range f.Decls {
				if fn, isFn := d.(*ast.FuncDecl); isFn {
					node, val, tag, ok := getFnTag(fn)
					fnt := fnTag{tag: tag, node: node, pkg: pack.Name, file: file, fn: fn.Name.Name}
					if ok {
						funcs = append(funcs, fnt)
						if *write {
							val.Value = `"` + fnt.expected() + `"`
						}
					}

				}
			}
		}
	}

	for _, fn := range funcs {
		if fn.tag != fn.expected() {
			fmt.Printf("%s:%d Incorrect fn tag %q. Should be %q\n", fn.file, set.Position(fn.node.Pos()).Line, fn.tag, fn.expected())
		}
	}

	if *write {
		for _, pack := range packs {
			for file, f := range pack.Files {
				var (
					newFName = file + ".new"
					oldFName = file + ".old"
				)
				newFile, err := os.Create(newFName)
				if err != nil {
					log.Fatalf("Error creating file %q", err)
				}

				err = format.Node(newFile, set, f)
				if err != nil {
					log.Fatalf("Error writing to file %q", err)
				}

				err = newFile.Close()
				if err != nil {
					log.Fatalf("Error closing file %q", err)
				}

				err = os.Rename(file, oldFName)
				if err != nil {
					log.Fatalf("Error renaming old file %q", err)
				}

				err = os.Rename(newFName, file)
				if err != nil {
					log.Fatalf("Error renaming new file %q", err)
				}

				err = os.Remove(oldFName)
				if err != nil {
					log.Fatalf("Error deleting new file %q", err)
				}
			}
		}
	}
}

func getFnTag(fn *ast.FuncDecl) (*ast.Ident, *ast.BasicLit, string, bool) {
	block := fn.Body
	for _, stmt := range block.List {
		ass, ok := stmt.(*ast.AssignStmt)
		if !ok {
			continue
		}

		if len(ass.Lhs) != 1 || len(ass.Rhs) != 1 {
			continue
		}

		variable, ok := ass.Lhs[0].(*ast.Ident)
		if !ok {
			continue
		}

		if variable.Name == "fn" {
			val, ok := ass.Rhs[0].(*ast.BasicLit)
			if !ok {
				continue
			}

			return variable, val, strings.Trim(val.Value, `"`), true
		}
	}
	return nil, nil, "", false
}
