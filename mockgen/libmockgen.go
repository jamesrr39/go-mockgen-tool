package mockgen

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

var ErrInterfaceTypeNotFound = errors.New("interface type not found")

type TypeData struct {
	PackageName        string
	Imports            []*ast.ImportSpec
	Methods            []Method
	EmbeddedInterfaces []string
}

type Type struct {
	PackageName, TypeName, Name string
}

type Method struct {
	Name        string
	Params      []Type
	ReturnTypes []Type
	Signature   string
}

func GetMethodsForType(sourceCode, interfaceName string) (*TypeData, error) {
	parsedFile, err := parser.ParseFile(token.NewFileSet(), "", sourceCode, 0)
	if err != nil {
		return nil, err
	}

	var itemNameFound bool
	var allDone bool

	typeData := &TypeData{
		PackageName: parsedFile.Name.Name,
		Methods:     nil,
	}

	importPathShortNames := make(map[string]struct{})

	ast.Inspect(parsedFile, func(node ast.Node) bool {
		if allDone {
			return false
		}
		switch n := node.(type) {
		case *ast.Ident:
			if n.Name == interfaceName {
				itemNameFound = true
			}
		case *ast.InterfaceType:
			if itemNameFound == false {
				// not the interface we are looking for, skip
				return false
			}

			for _, astField := range n.Methods.List {
				var names []string
				switch astFieldType := astField.Type.(type) {
				case *ast.SelectorExpr:
					// embedded interfaces in other packages, e.g. `type X interface {io.Reader}`
					importPathName := astFieldType.X.(*ast.Ident).Name
					if importPathName != "" {
						importPathShortNames[importPathName] = struct{}{}
					}
					name := getNameForAstNode(sourceCode, astFieldType)
					typeData.EmbeddedInterfaces = append(typeData.EmbeddedInterfaces, name)
				case *ast.Ident:
					// embedded interfaces in the same package
					typeData.EmbeddedInterfaces = append(typeData.EmbeddedInterfaces, astFieldType.Name)
				case *ast.FuncType:
					// functions defined on the interface
					for _, name := range astField.Names {
						names = append(names, name.String())
					}

					var paramTypes, returnTypes []Type

					switch t := astField.Type.(type) {
					case *ast.FuncType:
						for _, param := range t.Params.List {
							paramNameText := getNameForAstNode(sourceCode, param)
							paramTypesInMethod := getTypesFromText(paramNameText)

							for _, paramType := range paramTypesInMethod {
								if paramType.PackageName != "" {
									importPathShortNames[paramType.PackageName] = struct{}{}
									println("adding::", paramType.PackageName)
								}
							}

							paramTypes = append(paramTypes, paramTypesInMethod...)
							// println("pnt:", paramNameText)

							// for _, name := range param.Names {
							// 	paramNames = append(paramNames, name.Name)
							// }
						}
						if t.Results != nil {
							// for _, l := range t.Results.List {
							// 	println("names::", getNameForAstNode(sourceCode, l.Type))
							// 	for _, n := range l.Names {
							// 		println(n.Name, n.Obj)

							// 	}
							// 	returnTypes = append(returnTypes, getNameForAstNode(sourceCode, l))
							// }

							returnTypesFromMethods := getTypesFromText(getNameForAstNode(sourceCode, t.Results))

							for _, returnType := range returnTypesFromMethods {
								if returnType.PackageName != "" {
									importPathShortNames[returnType.PackageName] = struct{}{}
									println("adding::", returnType.PackageName)
								}
							}

							returnTypes = append(returnTypes, returnTypesFromMethods...)
						}
					}

					signature := getNameForAstNode(sourceCode, astField.Type)

					for _, name := range names {
						typeData.Methods = append(
							typeData.Methods,
							Method{
								name,
								paramTypes,
								returnTypes,
								signature,
							},
						)

					}
				}
			}

			allDone = true
			return false
		}

		return true
	})

	if !itemNameFound {
		return nil, ErrInterfaceTypeNotFound
	}

	// add imports
	for _, im := range parsedFile.Imports {
		pathFragments := strings.Split(im.Path.Value, "/")
		shortName := pathFragments[len(pathFragments)-1]
		if im.Name != nil {
			shortName = im.Name.Name
		}
		shortName = strings.Trim(shortName, `"`)
		println("checking::", shortName)
		_, ok := importPathShortNames[shortName]
		if !ok {
			// not required
			continue
		}
		typeData.Imports = append(typeData.Imports, im)
	}

	return typeData, nil
}

// func namesToTypes(names []string) []Type {
// 	var types []Type
// 	for _, name := range names {
// 		fragments := strings.Split(name, ".")
// 		switch len(fragments) {
// 		case 1:
// 			types = append(types, Type{Name: fragments[0]})
// 		case 2:
// 			types = append(types, Type{PackageName: fragments[0], Name: fragments[1]})

// 		default:
// 			panic(fmt.Sprintf("unexpected number of fragments (%d) in name: %q", len(fragments), name))
// 		}
// 	}
// 	return types
// }

func getTypesFromText(str string) []Type {
	// e.g.
	// err1, err2 extrapkg.Error
	// mode, mode2 DriveMode
	// DriveMode
	// count int
	fragments := strings.Split(str, " ")
	switch len(fragments) {
	case 0:
		panic("unexpected 0 length fragments: " + str)
	default:
		fullTypeName := fragments[len(fragments)-1]
		fullTypeNameFragments := strings.Split(fullTypeName, ".")
		packageName := ""
		typeName := fullTypeNameFragments[0]
		if len(fullTypeNameFragments) > 1 {
			packageName = fullTypeNameFragments[0]
			typeName = fullTypeNameFragments[1]
		}
		if len(fragments) == 1 {
			// just one, unnamed type
			return []Type{{
				PackageName: packageName,
				TypeName:    typeName,
			}}
		}

		var types []Type
		for _, f := range fragments[:len(fragments)-1] {
			types = append(types, Type{
				PackageName: packageName,
				TypeName:    typeName,
				Name:        strings.Trim(f, ","),
			})
		}
		return types
	}
}

func WriteMockType(interfaceName string, typeData *TypeData) string {
	packageDef := fmt.Sprintf("// Code generated by go-mockgen-tool: https://github.com/jamesrr39/go-mockgen-tool. DO NOT EDIT.\n\npackage %s\n\n", typeData.PackageName)

	return packageDef + createImportsDef(typeData) + createStructDef(typeData, interfaceName) + createMethodsDef(typeData, interfaceName)
}

func createImportsDef(typeData *TypeData) string {
	var importsDef string
	if len(typeData.Imports) > 0 {
		importsDef += "import (\n"
		for _, im := range typeData.Imports {
			importDef := "\t"
			if im.Name != nil {
				importDef += fmt.Sprintf("%s ", im.Name.Name)
			}

			importDef += im.Path.Value

			importsDef += importDef + "\n"
		}
		importsDef += ")\n\n"
	}
	return importsDef
}

func createStructDef(typeData *TypeData, interfaceName string) string {
	structDef := fmt.Sprintf("type Mock%s struct {\n", interfaceName)
	var longestMethodNameLen int
	for _, method := range typeData.Methods {
		if len(method.Name) > longestMethodNameLen {
			longestMethodNameLen = len(method.Name)
		}
	}
	methodMaxLengthForPadding := longestMethodNameLen + len(internalFuncSuffix)
	// right-pad the method names to line them up
	methodNameTemplate := fmt.Sprintf("\t%%-%ds func%%s\n", methodMaxLengthForPadding)
	for _, method := range typeData.Methods {
		methodName := method.Name + internalFuncSuffix
		structDef += fmt.Sprintf(methodNameTemplate, methodName, method.Signature)
	}
	for _, embeddedInterface := range typeData.EmbeddedInterfaces {
		structDef += fmt.Sprintf("\t%s\n", embeddedInterface)
	}
	structDef += fmt.Sprintln("}")
	return structDef
}

func createMethodsDef(typeData *TypeData, interfaceName string) string {
	var methodsDef string
	for _, method := range typeData.Methods {
		hasReturn := len(method.ReturnTypes) != 0

		returnKeywordText := ""
		if hasReturn {
			returnKeywordText = "return "
		}

		var paramNames []string
		for i, param := range method.Params {
			paramName := param.Name
			if paramName == "" {
				paramName = fmt.Sprintf("param%d", i)
			}
			paramNames = append(paramNames, paramName)
		}

		methodsDef += fmt.Sprintf(`
func (o *Mock%s) %s%s {
	if o.%sFunc == nil {
		panic("%sFunc not defined")
	}
	%so.%s%s(%s)
}
`, interfaceName, method.Name, method.Signature,
			method.Name,
			method.Name,
			returnKeywordText, method.Name, internalFuncSuffix, strings.Join(paramNames, ", "))
	}
	return methodsDef
}

func getNameForAstNode(sourceCode string, node ast.Node) string {
	return sourceCode[node.Pos()-1 : node.End()-1]
}

const internalFuncSuffix = "Func"
