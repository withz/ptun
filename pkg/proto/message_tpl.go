package proto

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"text/template"
)

const genCodeFileName = "init.gen.go"

func RegisterMessage(t reflect.Type) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	messageRegistry[t.Name()] = t
}
func GenMessageMethod(dir string) (err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() == genCodeFileName {
			continue
		}
		if filepath.Ext(entry.Name()) == ".go" {
			files = append(files, path.Join(dir, entry.Name()))
		}
	}

	fset := token.NewFileSet()
	out, err := os.OpenFile(path.Join(dir, genCodeFileName), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	tmpl, err := template.New("header").Parse(templateHeader)
	if err != nil {
		return err
	}

	tmplRow, err := template.New("row").Parse(templateRow)
	if err != nil {
		return err
	}

	headerSetted := false

	for _, f := range files {
		node, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		pkgName := node.Name.Name
		if !headerSetted {
			headerSetted = true
			data := struct{ Package string }{
				Package: pkgName,
			}
			err = tmpl.Execute(out, data)
			if err != nil {
				return err
			}
		}

		for _, decl := range node.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structName := typeSpec.Name.Name

				data := struct{ Name string }{
					Name: structName,
				}
				err = tmplRow.Execute(out, data)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

const templateHeader = `// Code generated .* DO NOT EDIT

package {{.Package}}

import (
	"reflect"

	"github.com/withz/ptun/pkg/proto"
)
`

const templateRow = `
func init() {
	proto.RegisterMessage(reflect.TypeFor[{{.Name}}]())
}
`
