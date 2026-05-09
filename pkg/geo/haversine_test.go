package geo

import (
	"math"
	"testing"
)

func TestHaversine_ZeroDistance(t *testing.T) {
	got := Haversine(-6.2088, 106.8456, -6.2088, 106.8456)
	if got != 0 {
		t.Errorf("expected 0, got %f", got)
	}
}

func TestHaversine_KnownDistance(t *testing.T) {
	// Monas (Jakarta) → Bundaran HI ≈ 1.5 km
	monasLat, monasLng := -6.1754, 106.8272
	hiLat, hiLng := -6.1944, 106.8230
	got := Haversine(monasLat, monasLng, hiLat, hiLng)
	// Allow ±200 m tolerance
	if math.Abs(got-2150) > 300 {
		t.Errorf("expected ~2150m, got %f", got)
	}
}
