package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

//Шаблон, на основе которого будем генерировать
var repositoryTemplate = template.Must(template.New("").Parse(`
package main

import (
    "github.com/jinzhu/gorm"
)

type {{ .EntityName }}Repository struct {
	db *gorm.DB
}

func New{{ .EntityName }}Repository(db *gorm.DB) {{ .EntityName }}Repository {
	return {{ .EntityName }}Repository{ db: db }
}

func (r {{ .EntityName }}Repository) Get({{ .PrimaryName }} {{ .PrimaryType }}) (*{{ .EntityName }}, error) {
entity := new({{ .EntityName }})
	err := r.db.Limit(1).Where("{{ .PrimarySQLName }} = ?", {{ .PrimaryName }}).Find(entity).Error
return entity, err
}

func (r {{ .EntityName }}Repository) Create(entity *{{ .EntityName }}) error {
	return r.db.Create(entity).Error
}

func (r {{ .EntityName }}Repository) Update(entity *{{ .EntityName }}) error {
	return r.db.Model(entity).Update(entity).Error
}

func (r {{ .EntityName }}Repository) Delete(entity *{{ .EntityName }}) error {
	return r.db.Delete(entity).Error
}
`))

//Агрегатор данных для установки параметров в шаблоне
type repositoryGenerator struct {
	typeSpec   *ast.TypeSpec
	structType *ast.StructType
}

func (r *repositoryGenerator) Generate(file *ast.File) error {
	//Находим первичный ключ
	primary, err := r.primaryField()
	if err != nil {
		return err
	}

	type templateParams struct {
		EntityName     string
		PrimaryType    string
		PrimaryName    string
		PrimarySQLName string
	}
	//Аллокация и установка параметров для template
	params := templateParams{
		EntityName:     r.typeSpec.Name.Name,
		PrimaryName:    strcase.ToLowerCamel(primary.Names[0].Name),
		PrimarySQLName: strcase.ToSnake(primary.Names[0].Name),
		PrimaryType:    expr2string(primary.Type),
	}
	//Аллокация буфера,
	//куда будем заливать выполненный шаблон
	var buf bytes.Buffer
	//Процессинг шаблона с подготовленными параметрами
	//в подготовленный буфер
	err = repositoryTemplate.Execute(&buf, params)
	if err != nil {
		return fmt.Errorf("execute template: %v", err)
	}
	//Парсинг обработанного шаблона,
	//который уже стал валидным кодом Go,
	//в дерево разбора,
	//получаем AST этого кода
	templateAst, err := parser.ParseFile(
		token.NewFileSet(),
		//Источник для парсинга лежит не в файле,
		"",
		//а в буфере
		buf.Bytes(),
		parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse template: %v", err)
	}
	//Добавляем декларации из полученного дерева
	//в результирующий outFile *ast.File,
	for _, decl := range templateAst.Decls {
		file.Decls = append(file.Decls, decl)
	}
	return nil
}

//Ищем то, что мы пометили gorm:"primary_key"
func (r repositoryGenerator) primaryField() (*ast.Field, error) {
	for _, field := range r.structType.Fields.List {
		if !strings.Contains(field.Tag.Value, "primary") {
			continue
		}
		return field, nil
	}

	return nil, fmt.Errorf("has no primary field")
}
