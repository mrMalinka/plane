package gps

const (
	PrefixGGA = "$GNGGA"
	PrefixRMC = "$GNRMC"
)

const (
	FixQualityInvalid = "0" // no fix
	FixQualityGPS     = "1" // gps fix
	FixQualityDGPS    = "2" // differential gps fix
)

const (
	RMCValidityValid   = 'A'
	RMCValidityInvalid = 'V'
)
