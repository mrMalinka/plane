module pico
//
go 1.23.8
//
require (
//	device v0.0.0-00010101000000-000000000000 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
//	machine v0.0.0-00010101000000-000000000000
//	runtime/interrupt v0.0.0-00010101000000-000000000000 // indirect
//	runtime/volatile v0.0.0-00010101000000-000000000000 // indirect
	tinygo.org/x/drivers v0.32.0
)
//
replace (
//	device => ./modules/device
//	machine => ./modules/machine
//
//	runtime => ./modules/runtime
//	runtime/interrupt => ./modules/runtime/interrupt
//	runtime/volatile => ./modules/runtime/volatile
)
