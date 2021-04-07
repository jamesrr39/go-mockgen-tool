# go-mockgen

go-mockgen is a package that creates mock object definitions based on an interface.

It uses code generation to generate a mock with solid types (no reflection or interface{} needed!)

Installation with: `go get github.com/jamesrr39/go-mockgen-tool`.

You can create a mock by using the `go:generate go-mockgen-tool --type <my type name>`, or simply by running the go-mockgen-tool inside the package directory. You must specify the name of the type you want to create a definition for.

For an example, see the [example folder](./example).
