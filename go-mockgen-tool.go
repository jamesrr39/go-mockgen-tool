package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/jamesrr39/go-mockgen-tool/mockgen"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	var interfaceName, outFilePath string
	kingpin.Flag("type", "name of the interface type").Required().StringVar(&interfaceName)
	kingpin.Flag("o", "out file. File to write the generated type to. Defaults to <typename>_mock.go").StringVar(&outFilePath)
	kingpin.Parse()

	fileInfos, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatalf("error reading directory: %q", err)
	}

	var typeData *mockgen.TypeData
	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() || !strings.HasSuffix(fileInfo.Name(), ".go") {
			// skip non Go files
			continue
		}

		sourceCode, err := ioutil.ReadFile(fileInfo.Name())
		if err != nil {
			log.Fatalf("couldn't read %q. Error: %q", fileInfo.Name(), err)
		}

		typeData, err = mockgen.GetMethodsForType(string(sourceCode), interfaceName)
		if err != nil {
			if err != mockgen.ErrInterfaceTypeNotFound {
				log.Fatalf("error generating mock: %s\n", err)
			}
			// continue with other files
			continue
		}

		break
	}

	if typeData == nil {
		log.Fatalln("no methods found. Either there were no files with the interface name or there were no methods to mock")
	}

	mockText := mockgen.WriteMockType(interfaceName, typeData)

	if outFilePath == "" {
		outFilePath = fmt.Sprintf("%s_mock.go", strings.ToLower(interfaceName))
	}

	err = ioutil.WriteFile(outFilePath, []byte(mockText), 0664)
	if err != nil {
		log.Fatalf("error writing mock to %q: %s\n", outFilePath, err)
	}
}
