package example

import (
	"io"
	"os"
	osfs "os"

	"github.com/jamesrr39/go-mockgen-tool/example/extrapkg"
)

type DriveMode int

const (
	DriveModeFast DriveMode = iota
)

//go:generate go-mockgen-tool --type Vehicle
type Vehicle interface {
	Name() string
	WheelCount() (int, error)
	test2(mode, mode2 DriveMode) func(cargoWeightKg float64) (float64, error)
	GetReader() io.Reader
	// DoSomething is a no-return function
	DoSomething()
	DoSomething2(err1, err2 extrapkg.Error, a int)
	DoSomething3(extrapkg.Error, int, func(a, b string) extrapkg.Error)
	io.Writer
	// SecondInterface is an interface in the same package
	SecondInterface
	osfs.Signal
}

type SecondInterface interface {
	os.FileInfo
	io.WriteCloser
}
