// Package geo provides geographic distance helpers.
package geo

import "math"

const earthRadiusMeters = 6371000.0

// Haversine returns the great-circle distance between two points in metres.
// Inputs are latitude / longitude in degrees.
func Haversine(lat1, lng1, lat2, lng2 float64) float64 {
	rad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := rad(lat2 - lat1)
	dLng := rad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeters * c
}
