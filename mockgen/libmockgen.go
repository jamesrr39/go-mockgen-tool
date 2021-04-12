package mockgen

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

var ErrInterfaceTypeNotFound = errors.New("interface type not found")

const internalFuncSuffix = "Func"

type TypeData struct {
	PackageName        string
	Imports            []*ast.ImportSpec
	Methods            []Method
	EmbeddedInterfaces []string
}

type Type struct {
	PackageName, TypeName, Name string
}

func (t Type) FullTypeName() string {
	if t.PackageName == "" {
		return t.TypeName
	}

	return fmt.Sprintf("%s.%s", t.PackageName, t.TypeName)
}

type Method struct {
	Name        string
	Params      []Type
	ReturnTypes []Type
}

func (method Method) ParamNames() []string {
	var paramNames []string
	for i, param := range method.Params {
		paramName := param.Name
		if paramName == "" {
			paramName = fmt.Sprintf("param%d", i)
		}
		paramNames = append(paramNames, paramName)
	}

	return paramNames
}

func (method Method) ParamsWithTypes() string {
	var fullParamFragments []string
	for i, param := range method.Params {
		paramName := param.Name
		if paramName == "" {
			paramName = fmt.Sprintf("param%d", i)
		}
		fullParamFragments = append(fullParamFragments, fmt.Sprintf("%s %s", paramName, param.FullTypeName()))
	}

	return strings.Join(fullParamFragments, ", ")
}

func (method Method) ReturnTypesAsString() string {
	var returnFragments []string
	for _, ret := range method.ReturnTypes {
		returnFragments = append(returnFragments, ret.FullTypeName())
	}

	var retSignature string
	switch len(returnFragments) {
	case 0:
		// do nothing, should be empty string
	case 1:
		retSignature = fmt.Sprintf("%s ", returnFragments[0])
	default:
		retSignature = fmt.Sprintf("(%s) ", strings.Join(returnFragments, ", "))
	}

	return retSignature
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
								}
							}

							paramTypes = append(paramTypes, paramTypesInMethod...)
						}
						if t.Results != nil {
							retText := getNameForAstNode(sourceCode, t.Results)
							if strings.HasPrefix(retText, "(") {
								retText = strings.TrimPrefix(retText, "(")
								retText = strings.TrimSuffix(retText, ")")
							}
							returnTypesFromMethods := getTypesFromText(retText)

							for _, returnType := range returnTypesFromMethods {
								if returnType.PackageName != "" {
									importPathShortNames[returnType.PackageName] = struct{}{}
								}
							}

							returnTypes = append(returnTypes, returnTypesFromMethods...)
						}
					}

					for _, name := range names {
						typeData.Methods = append(
							typeData.Methods,
							Method{
								name,
								paramTypes,
								returnTypes,
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
		_, ok := importPathShortNames[shortName]
		if !ok {
			// not required
			continue
		}
		typeData.Imports = append(typeData.Imports, im)
	}

	return typeData, nil
}

type shortPackageNameMapType map[string]struct{}

func currentTokenToParamsObjects(text string) []string {
	text = strings.TrimSpace(text)

	if funcDefRegex.MatchString(text) {
		return []string{text}
	}

	// not func
	spaceIdx := strings.Index(text, " ")
	if spaceIdx == -1 {
		return []string{text}
	}
	paramName := strings.TrimSpace(text[:spaceIdx])
	typeName := strings.TrimSpace(text[spaceIdx:])

	return []string{paramName, typeName}
}

func fullTypeToType(fullType string) Type {
	if funcDefRegex.MatchString(fullType) {
		return Type{TypeName: fullType}
	}

	dotIndex := strings.Index(fullType, ".")
	if dotIndex == -1 {
		return Type{TypeName: fullType}
	}

	return Type{
		PackageName: fullType[:dotIndex],
		TypeName:    fullType[dotIndex+1:],
	}
}

func parseParams(str string) [][]string {
	currentToken := new(currentTokenType)
	var paramsObjects [][]string
	var funcNestingLevel int

	for _, c := range str {
		switch c {
		case ',':
			if funcNestingLevel == 0 {
				// if not 0, we are not finished with this yet. We either have no current token or we are in a function definition
				paramsObjects = append(paramsObjects, currentTokenToParamsObjects(currentToken.Token))

				// reset
				currentToken = new(currentTokenType)
			}
		case '(':
			funcNestingLevel++
		case ')':
			funcNestingLevel--
		}

		if funcNestingLevel == 0 && c == ',' {
		} else {
			currentToken.Token += string(c)
		}
	}

	if funcNestingLevel > 0 {
		panic("there was an unclosed function in: " + str)
	}

	// add remaining one to end
	paramsObjects = append(paramsObjects, currentTokenToParamsObjects(currentToken.Token))

	return paramsObjects
}

type currentTokenType struct {
	Token string
}

var funcDefRegex = regexp.MustCompile(`^func\s*\(.*`)

func getTypesFromText(str string) []Type {
	// e.g.
	// err1, err2 extrapkg.Error
	// mode, mode2 DriveMode
	// DriveMode
	// count int
	// func(int, func(int, int))

	paramsObjects := parseParams(str)

	if len(paramsObjects) == 0 {
		panic("unexpected 0 length fragments: " + str)
	}

	lastFragment := paramsObjects[len(paramsObjects)-1]
	if len(lastFragment) == 1 {
		// if the last parameter doesn't have a name, the definition only contains types, not names
		var types []Type
		for _, fragment := range paramsObjects {
			types = append(types, fullTypeToType(fragment[0]))
		}
		return types
	}

	// definition contains named types
	var previousType *Type
	var types []Type
	for i := len(paramsObjects) - 1; i >= 0; i-- {
		paramFragments := paramsObjects[i]
		var t Type
		switch len(paramFragments) {
		case 1:
			t = *previousType
			t.Name = paramFragments[0]
		case 2:
			t = fullTypeToType(paramFragments[1])
			t.Name = paramFragments[0]
		default:
			panic(fmt.Sprintf("found unexpected number of fragments: %d", len(paramFragments)))
		}

		types = append(types, t)
		previousType = &t
	}

	return reverseTypesSlice(types)
}

func reverseTypesSlice(in []Type) []Type {
	out := make([]Type, len(in))
	for i := len(in) - 1; i >= 0; i-- {
		outPos := (len(in) - 1) - i
		out[outPos] = in[i]
	}
	return out
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
		signature := fmt.Sprintf("(%s) %s", method.ParamsWithTypes(), method.ReturnTypesAsString())
		structDef += fmt.Sprintf(methodNameTemplate, methodName, signature)
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

		var fullParamFragments []string
		var paramNames []string
		for i, param := range method.Params {
			paramName := param.Name
			if paramName == "" {
				paramName = fmt.Sprintf("param%d", i)
			}
			paramNames = append(paramNames, paramName)
			fullParamFragments = append(fullParamFragments, fmt.Sprintf("%s %s", paramName, param.FullTypeName()))
		}

		var returnFragments []string
		for _, ret := range method.ReturnTypes {
			returnFragments = append(returnFragments, ret.FullTypeName())
		}

		var retSignature string
		switch len(returnFragments) {
		case 0:
			// do nothing, should be empty string
		case 1:
			retSignature = fmt.Sprintf("%s ", returnFragments[0])
		default:
			retSignature = fmt.Sprintf("(%s) ", strings.Join(returnFragments, ", "))
		}

		methodsDef += fmt.Sprintf(`
func (o *Mock%s) %s(%s) %s{
	if o.%sFunc == nil {
		panic("%sFunc not defined")
	}
	%so.%s%s(%s)
}
`, interfaceName, method.Name, strings.Join(fullParamFragments, ", "), retSignature,
			method.Name,
			method.Name,
			returnKeywordText, method.Name, internalFuncSuffix, strings.Join(paramNames, ", "))
	}
	return methodsDef
}

func getNameForAstNode(sourceCode string, node ast.Node) string {
	return sourceCode[node.Pos()-1 : node.End()-1]
}
