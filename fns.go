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
	tag        string
	node       *ast.Ident
	valNode    *ast.BasicLit
	pkg        string
	file       string
	fn         string
	methodType string
}

func (t *fnTag) oldStyle() string {
	fileName := filepath.Base(t.file)
	extension := filepath.Ext(fileName)
	fileName = fileName[0 : len(fileName)-len(extension)]
	return fmt.Sprintf("%s.%s.%s", t.pkg, fileName, t.fn)
}

func (t *fnTag) newStyle() string {
	if t.methodType == "" {
		return t.oldStyle()
	}
	fileName := filepath.Base(t.file)
	extension := filepath.Ext(fileName)
	fileName = fileName[0 : len(fileName)-len(extension)]
	return fmt.Sprintf("%s.%s.%s-%s", t.pkg, fileName, t.methodType, t.fn)
}

func (t *fnTag) needsReplacing(tag string) (replacement string) {
	if tag == t.oldStyle() {
		return ""
	}
	n := t.newStyle()
	if tag == n {
		return ""
	}
	return n
}

func cleanPath(path string) string {
	cln := filepath.Clean(path)
	if filepath.IsAbs(path) || strings.HasPrefix(cln, ".") {
		return cln
	}
	return "." + string(os.PathSeparator) + cln
}

func main() {
	write := flag.Bool("w", false, "Rewrite the files with correct fn tags")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Println("Usage fns <package>")
		os.Exit(1)
	}

	pkg := flag.Args()[0]

	// TODO: allow ./... format
	set := token.NewFileSet()
	stat, err := os.Stat(pkg)
	if err != nil {
		fmt.Println("Failed to stat file or dir:", err)
		os.Exit(1)
	}
	type file struct {
		name string
		file *ast.File
	}
	var files []file
	switch {
	case stat.IsDir():
		packs, err := parser.ParseDir(set, pkg, nil, parser.ParseComments)
		if err != nil {
			fmt.Println("Failed to parse package:", err)
			os.Exit(1)
		}
		for _, pack := range packs {
			for name, f := range pack.Files {
				files = append(files, file{cleanPath(name), f})
			}
		}
	default:
		// TODO: glob?
		f, err := parser.ParseFile(set, pkg, nil, parser.ParseComments)
		if err != nil {
			fmt.Println("Failed to parse file:", err)
			os.Exit(1)
		}
		files = []file{{cleanPath(pkg), f}}
	}

	funcs := []fnTag{}
	for _, f := range files {

		for _, d := range f.file.Decls {
			if fn, isFn := d.(*ast.FuncDecl); isFn {
				node, val, tag, ok := getFnTag(fn)
				fnt := fnTag{tag: tag, node: node, pkg: f.file.Name.Name, valNode: val, file: f.name, fn: fn.Name.Name}
				if ok {
					if fn.Recv != nil {
						if len(fn.Recv.List) != 1 {
							fmt.Fprintf(os.Stderr, "Method found with more than one receiver")
							os.Exit(1)
						} // if
						switch tt := fn.Recv.List[0].Type.(type) {
						case *ast.Ident:
							fnt.methodType = tt.Name
						case *ast.StarExpr:
							fnt.methodType = "*" + tt.X.(*ast.Ident).Name
						default:
							fmt.Fprintf(os.Stderr, "Invalid receiver type %T", tt)
							os.Exit(1)
						}
					}
					funcs = append(funcs, fnt)
				}

			}
		}
	}

	for _, fn := range funcs {
		if r := fn.needsReplacing(fn.tag); r != "" {
			fmt.Printf("%s:%d Incorrect fn tag %q. Should be %q\n", fn.file, set.Position(fn.node.Pos()).Line, fn.tag, r)
			if *write {
				fn.valNode.Value = `"` + r + `"`
			}
		}
	}

	if *write {
		for _, f := range files {
			var (
				newFName = f.name + ".new"
				oldFName = f.name + ".old"
			)
			newFile, err := os.Create(newFName)
			if err != nil {
				log.Fatalf("Error creating file %q", err)
			}

			err = format.Node(newFile, set, f.file)
			if err != nil {
				log.Fatalf("Error writing to file %q", err)
			}

			err = newFile.Close()
			if err != nil {
				log.Fatalf("Error closing file %q", err)
			}

			err = os.Rename(f.name, oldFName)
			if err != nil {
				log.Fatalf("Error renaming old file %q", err)
			}

			err = os.Rename(newFName, f.name)
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
