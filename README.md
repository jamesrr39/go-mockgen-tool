# go-mockgen-tool

go-mockgen-tool is a package that creates mock object definitions based on an interface.

It uses code generation to generate a mock with solid types (no reflection or interface{} needed!). The resulting mock type has properties that you can fill in to determine how the function behaves.

Installation with: `go get github.com/jamesrr39/go-mockgen-tool`.

You can create a mock by using the `go:generate go-mockgen-tool --type <my type name>`, or simply by running the go-mockgen-tool inside the package directory. You must specify the name of the type you want to create a definition for.

With an interface with the following definition:

```
//go:generate go-mockgen-tool --type Vehicle
type Vehicle interface {
	WheelCount() (int, error)
	GetDriveFunc() func() error
}
```

go-mockgen-tool generates the following code:

```
type MockVehicle struct {
	WheelCountFunc   func() (int, error) 
	GetDriveFuncFunc func() func() error 
}

func (o *MockVehicle) WheelCount() (int, error) {
	if o.WheelCountFunc == nil {
		panic("WheelCountFunc not defined")
	}
	return o.WheelCountFunc()
}

func (o *MockVehicle) GetDriveFunc() func() error {
	if o.GetDriveFuncFunc == nil {
		panic("GetDriveFuncFunc not defined")
	}
	return o.GetDriveFuncFunc()
}
```

The "Mock" struct can then be supplied with custom functions to return the objects you wish during test runs (this tool is intended for use in tests, but not limited to use in tests).

For a more detailed example, see the [example folder](./example). It shows an interface with a variety of different return types, embedded interfaces etc, and the generated code.

Features supported:

- Functions with and without parameters and return types
- Functions with more complex parameters and return types, e.g. functions that return functions
- Embedded interfaces both in the same package and different packages
- Package aliasing

This probably don't support _every_ way to declare an interface. If you find something that doesn't work, but is valid Go, please open an issue.

### Related projects

- https://github.com/rjeczalik/interfaces: generate an interface from a given type
