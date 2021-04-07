package example

type MockVehicle struct {
	NameFunc func() string
	WheelCountFunc func() int
	FuelEffiencyFuncFunc func(mode DriveMode) func(cargoWeightKg float64) (float64, error)
	test2Func func(mode, mode2 DriveMode) func(cargoWeightKg float64) (float64, error)
}

func(o *MockVehicle) Name() string {
	if o.NameFunc == nil {
		panic("o.NameFunc not defined")
	}
	return o.NameFunc()
}

func(o *MockVehicle) WheelCount() int {
	if o.WheelCountFunc == nil {
		panic("o.WheelCountFunc not defined")
	}
	return o.WheelCountFunc()
}

func(o *MockVehicle) FuelEffiencyFunc(mode DriveMode) func(cargoWeightKg float64) (float64, error) {
	if o.FuelEffiencyFuncFunc == nil {
		panic("o.FuelEffiencyFuncFunc not defined")
	}
	return o.FuelEffiencyFuncFunc(mode)
}

func(o *MockVehicle) test2(mode, mode2 DriveMode) func(cargoWeightKg float64) (float64, error) {
	if o.test2Func == nil {
		panic("o.test2Func not defined")
	}
	return o.test2Func(mode, mode2)
}
