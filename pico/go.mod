module pico

go 1.24.4

require (
	device v0.0.0 // indirect
	machine v0.0.0
)

require (
	runtime/interrupt v0.0.0-00010101000000-000000000000 // indirect
	runtime/volatile v0.0.0-00010101000000-000000000000 // indirect
)

replace (
	device => ./modules/device
	machine => ./modules/machine

	runtime => ./modules/runtime
	runtime/interrupt => ./modules/runtime/interrupt
	runtime/volatile => ./modules/runtime/volatile
)
