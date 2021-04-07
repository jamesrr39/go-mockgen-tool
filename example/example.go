package example

type DriveMode int

const (
	DriveModeFast DriveMode = iota
)

//go:generate go-mockgen-tool --type Vehicle
type Vehicle interface {
	Name() string
	WheelCount() int
	FuelEffiencyFunc(mode DriveMode) func(cargoWeightKg float64) (float64, error)
	test2(mode, mode2 DriveMode) func(cargoWeightKg float64) (float64, error)
}
