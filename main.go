package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

var (
	gotypeRe = regexp.MustCompile(`^GoType: ?(.*)$`)
)

func main() {
	var (
		s string
		t string
	)
	flag.StringVar(&s, "s", "", "graphql sdl file path")
	flag.StringVar(&t, "t", "", "template file path")
	flag.Parse()
	if s == "" || t == "" {
		flag.Usage()
		return
	}
	sb, err := read(s)
	if err != nil {
		log.Fatal(err)
	}
	body, err := loadGQL(sb)
	if err != nil {
		log.Fatal(err)
	}

	tb, err := read(t)
	if err != nil {
		log.Fatal(err)
	}
	funcMap := sprig.GenericFuncMap()
	// TODO for cloud spanner type...
	funcMap["GoType"] = func(f *ast.FieldDefinition) string {
		switch f.Type.NamedType {
		case "ID", "String", "Int", "Float", "Boolean":
			return addPtPrefixIfNull(f.Type) + goSingleType(f.Type, body)
		case "": //list
			return "[]" + addPtPrefixIfNull(f.Type.Elem) + goSingleType(f.Type.Elem, body)
		default: // custom scalar, other object
			return addPtPrefixIfNull(f.Type) + goSingleType(f.Type, body)
		}
		return ""
	}
	tpl := template.Must(template.New(t).Funcs(template.FuncMap(funcMap)).Parse(string(tb)))
	tpl.Execute(os.Stdout, *body)
}
func read(file string) ([]byte, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func addPtPrefixIfNull(t *ast.Type) string {
	if t.NonNull {
		return ""
	}
	return "*"
}

func goSingleType(t *ast.Type, body *ast.Schema) string {
	switch t.NamedType {
	case "ID", "String":
		return "string"
	case "Int":
		return "int64"
	case "Float":
		return "float64"
	case "Boolean":
		return "bool"
	default:
		if t, ok := body.Types[t.NamedType]; ok {
			match := gotypeRe.FindStringSubmatch(t.Description)
			if match != nil && len(match) > 1 {
				return match[1]
			}
		}
		return t.NamedType
	}
	return ""
}

func loadGQL(b []byte) (*ast.Schema, error) {
	astDoc, err := gqlparser.LoadSchema(&ast.Source{
		Input: string(b),
	})
	if err != nil {
		return nil, err
	}
	return astDoc, nil
}
