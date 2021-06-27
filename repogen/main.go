package main

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"strings"
)

func main() {
	//Цель генерации передаётся переменной окружения
	path := os.Getenv("GOFILE")
	if path == "" {
		log.Fatal("GOFILE env variable must be set")
	}
	//Разбираем целевой файл в AST
	//Нас интересуют комментарии
	astInFile, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("parse file: %v", err)
	}
	//Список заданий генерации
	var tasks []repositoryGenerator
	//Запускаем инспектор и ищем деклараций
	ast.Inspect(astInFile, func(node ast.Node) bool {
		// нам нужны только декларации
		genDecl, ok := node.(*ast.GenDecl)
		if !ok {
			return true
		}
		//Код без комментариев не нужен,
		if genDecl.Doc == nil {
			return false
		}
		//интересуют спецификации типов,
		typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
		if !ok {
			return false
		}
		//а конкретно структуры
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return false
		}
		//Из оставшегося
		for _, comment := range genDecl.Doc.List {
			switch comment.Text {
			//выделяем структуры, помеченные комментарием repogen:entity,
			case "//repogen:entity":
				//и добавляем в список заданий генерации
				tasks = append(tasks, repositoryGenerator{
					typeSpec:   typeSpec,
					structType: structType,
				})
			}
		}

		return true
	})

	//Аллокация результирующего дерева разбора
	astOutFile := &ast.File{
		Name: astInFile.Name,
	}
	//Запускаем список заданий генерации
	for _, g := range tasks {
		//Для каждого задания вызываем генератор
		//Сгенерированные декларации помещаются в результирующее дерево разбора
		err = g.Generate(astOutFile)
		if err != nil {
			log.Fatalf("generate: %v", err)
		}
	}
	//Файл конечного результата всей работы,
	outFile, err := os.Create(strings.TrimSuffix(path, ".go") + "_gen.go")
	if err != nil {
		log.Fatalf("create file: %v", err)
	}
	defer outFile.Close()
	//Так сказать «печатаем» результирующий AST в результирующий файл исходного кода
	err = printer.Fprint(outFile, token.NewFileSet(), astOutFile)
	if err != nil {
		log.Fatalf("print file: %v", err)
	}
}
