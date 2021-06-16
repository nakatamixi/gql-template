package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/iancoleman/strcase"
	"github.com/jinzhu/inflection"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

var (
	gotypeRe     = regexp.MustCompile(`^GoType: ?(.*)$`)
	spanColumnRe = regexp.MustCompile(`^SpannerColumn: ?(.*)$`)
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
	// sprig camelcase (xstrings.ToCamelCase) is not valid
	funcMap["upperCamel"] = func(a string) string { return strcase.ToCamel(strings.ToLower(a)) }
	funcMap["joinstr"] = func(a, b string) string { return a + b }
	// TODO for cloud spanner type...
	funcMap["GoType"] = func(f *ast.FieldDefinition, replaceObjectType bool) string {
		switch f.Type.NamedType {
		case "ID", "String", "Int", "Float", "Boolean":
			return addPtPrefixIfNull(f.Type) + goSingleType(f.Type, body, replaceObjectType)
		case "": //list
			return "[]" + addPtPrefixIfNull(f.Type.Elem) + goSingleType(f.Type.Elem, body, replaceObjectType)
		default: // custom scalar, other object
			return addPtPrefixIfNull(f.Type) + goSingleType(f.Type, body, replaceObjectType)
		}
		return ""
	}
	funcMap["exists"] = func(d *ast.Definition, name string) bool {
		for _, it := range d.Fields {
			if strcase.ToCamel(it.Name) == name {
				return true
			}
		}
		return false

	}
	funcMap["foundPK"] = func(objName string, fields ast.FieldList) string {
		for _, f := range fields {
			desc := f.Description
			if strings.Contains(desc, "SpannerPK") || strcase.ToCamel(f.Name) == "Id" || strcase.ToCamel(f.Name) == strcase.ToCamel(objName+"Id") {
				return f.Name
			}

		}
		return ""
	}
	funcMap["ConvertObjectFieldName"] = func(f *ast.FieldDefinition) string {
		desc := f.Description
		match := spanColumnRe.FindStringSubmatch(desc)
		if match != nil && len(match) > 1 {
			return match[1]
		}
		namedType := f.Type.NamedType
		isArray := false
		if f.Type.NamedType == "" {
			isArray = true
			namedType = f.Type.Elem.NamedType
		}
		if def, ok := body.Types[namedType]; ok {
			if def.Kind == "OBJECT" {
				if isArray {
					return inflection.Plural(inflection.Singular(f.Name) + "Id")
				}
				return f.Name + "Id"
			} else {
				return f.Name
			}
		}
		return f.Name
	}
	tpl := template.Must(template.New(t).Funcs(template.FuncMap(funcMap)).Parse(string(tb)))
	if err := tpl.Execute(os.Stdout, *body); err != nil {
		log.Fatal(err)
	}
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

func goSingleType(t *ast.Type, body *ast.Schema, replace bool) string {
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
		if def, ok := body.Types[t.NamedType]; ok {
			match := gotypeRe.FindStringSubmatch(def.Description)
			if match != nil && len(match) > 1 {
				return match[1]
			}
			if !replace {
				return t.NamedType
			}
			for _, f := range def.Fields {
				desc := f.Description
				if strings.Contains(desc, "SpannerPK") || strcase.ToCamel(f.Name) == "Id" || strcase.ToCamel(f.Name) == strcase.ToCamel(t.NamedType+"Id") {
					return goSingleType(f.Type, body, replace)
				}

			}
			return "string"
		}
	}
	log.Fatalf("not found type %s", t.NamedType)
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
