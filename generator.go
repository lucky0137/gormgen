package gormgen

import (
	"bytes"
	"errors"
	"go/format"
	"io/ioutil"
	"log"
	"reflect"
	"strings"

	"github.com/jinzhu/gorm"
)

// fieldConfig
type fieldConfig struct {
	FieldName  string
	ColumnName string
	FieldType  string
	Tag        string
}

// structConfig
type structConfig struct {
	config
	StructName string
	Fields     []fieldConfig
}

type ImportPkg struct {
	Pkg string
}

type config struct {
	PkgName      string
	LogName      string
	ImportPkgs   []ImportPkg
	TransformErr bool
}

// The Generator is the one responsible for generating the code, adding the imports, formating, and writing it to the file.
type Generator struct {
	buf           map[string]*bytes.Buffer
	inputFile     string
	config        config
	structConfigs []structConfig
}

// NewGenerator function creates an instance of the generator given the name of the output file as an argument.
func NewGenerator(outputFile string) *Generator {
	return &Generator{
		buf:       map[string]*bytes.Buffer{},
		inputFile: outputFile,
	}
}

func (g *Generator) SetImportPkg(importPkgs []ImportPkg) *Generator {
	g.config.ImportPkgs = importPkgs
	return g
}

// SetPkgName
func (g *Generator) SetPkgName(name string) *Generator {
	g.config.PkgName = name
	return g
}

// SetLogName
func (g *Generator) SetLogName(logName string) *Generator {
	g.config.LogName = logName
	return g
}

// TransformError
func (g *Generator) TransformError() *Generator {
	g.config.TransformErr = true
	return g
}

// ParserStruct parse struct by reflect
func (g *Generator) ParserStruct(ptrs []interface{}) (ret *Generator) {
	for _, ptr := range ptrs {
		reType := reflect.TypeOf(ptr)
		if reType.Kind() != reflect.Ptr || reType.Elem().Kind() != reflect.Struct {
			panic("param dose't struct")
		}
		var (
			structData structConfig
			v          = reflect.ValueOf(ptr).Elem()
		)
		l := strings.Split(strings.Split(v.String(), " ")[0], ".")
		structData.StructName = l[len(l)-1]
		for i := 0; i < v.NumField(); i++ {
			var (
				field fieldConfig
			)
			structField := v.Type().Field(i)
			tag := structField.Tag
			tagValue := tag.Get("gorm")
			if strings.Contains(structField.Type.String(), ".Model") {
				field.FieldName = "ID"
				field.FieldType = "uint"
				field.ColumnName = gorm.ToDBName("ID")
			} else {
				if !strings.Contains(tagValue, "unique") && !strings.Contains(tagValue, "primary") {
					continue
				}
				field.FieldName = structField.Name
				field.FieldType = structField.Type.String()
				field.ColumnName = gorm.ToDBName(structField.Name)
			}

			structData.Fields = append(structData.Fields, field)
		}
		g.structConfigs = append(g.structConfigs, structData)
	}
	return g
}

// ParserAST parse by go file
func (g *Generator) ParserAST(p *Parser, structs []string) (ret *Generator) {
	for _, v := range structs {
		g.buf[v] = new(bytes.Buffer)
	}
	g.structConfigs = p.Parse()
	g.config.PkgName = p.pkg.Name
	return g
}

func (g *Generator) checkConfig() (err error) {
	if len(g.config.ImportPkgs) == 0 {
		err = errors.New("import package dose'n set")
		return
	}
	if len(g.config.PkgName) == 0 {
		err = errors.New("package name dose'n set")
		return
	}
	for i := 0; i < len(g.structConfigs); i++ {
		g.structConfigs[i].config = g.config
	}
	return
}

// Generate executes the template and store it in an internal buffer.
func (g *Generator) Generate() *Generator {
	if err := g.checkConfig(); err != nil {
		panic(err)
	}
	for _, v := range g.structConfigs {
		if err := outputTemplate.Execute(g.buf[v.StructName], v); err != nil {
			panic(err)
		}
	}

	return g
}

// Format function formates the output of the generation.
func (g *Generator) Format() *Generator {
	for k, _ := range g.buf {
		formatedOutput, err := format.Source(g.buf[k].Bytes())
		if err != nil {
			panic(err)
		}
		g.buf[k] = bytes.NewBuffer(formatedOutput)
	}
	return g
}

// Flush function writes the output to the output file.
func (g *Generator) Flush() error {
	for k, _ := range g.buf {
		if err := ioutil.WriteFile(g.inputFile+"/gen_"+strings.ToLower(k)+".go", g.buf[k].Bytes(), 0777); err != nil {
			log.Fatalln(err)
		}
	}
	return nil
}