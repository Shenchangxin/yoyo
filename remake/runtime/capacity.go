// Copyright 2026 Hypit.AI
// Translated from @hypit/runtime/src/capacity.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package runtime

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// CapacityResourceClaim is a shared resource.
// Absent PeriodMs means occupancy. A period means a replenishing rate budget.
type CapacityResourceClaim struct {
	ID string
	// Limit is the occupancy ceiling, or the burst size of a rate budget.
	Limit int64
	// Units is the amount held by this Command. Nil means one unit.
	Units *int64
	// PeriodMs replenishes Limit units per period. Nil means the property is absent.
	PeriodMs *int64
}

// CapacityUnits returns the units held by one claim.
func CapacityUnits(resource CapacityResourceClaim) int64 {
	units := int64(1)
	if resource.Units != nil {
		units = *resource.Units
	}
	if len(jsTrim(resource.ID)) == 0 || !safeInteger(resource.Limit) || resource.Limit < 1 || !safeInteger(units) || units < 1 {
		panic(errors.New("Capacity " + resource.ID + " requires a positive integer limit and units"))
	}
	if units > resource.Limit {
		panic(errors.New("Capacity " + resource.ID + " request needs " + strconv.FormatInt(units, 10) + " units but its limit is " + strconv.FormatInt(resource.Limit, 10)))
	}
	if resource.PeriodMs != nil && (!safeInteger(*resource.PeriodMs) || *resource.PeriodMs < 1) {
		panic(errors.New("Resource " + resource.ID + " periodMs must be a positive integer"))
	}
	return units
}

// CapacityReservation is capacity held by one executing Command.
type CapacityReservation struct {
	Build     string
	Command   string
	Resources []CapacityResourceClaim
	CreatedAt float64
	// Lifetime is "action" or "operation". Nil means the property is absent.
	Lifetime *string
}

// CapacityAcquire is "acquired" or "blocked".
type CapacityAcquire struct {
	Status      string
	Reservation *CapacityReservation
	// AvailableAt is absent when only a resource release can make this work ready.
	AvailableAt *float64
	Reason      string
	Resource    string
}

func safeInteger(value int64) bool {
	return value >= -maxSafeInteger && value <= maxSafeInteger
}

func safeNumber(value float64) bool {
	if math.IsNaN(value) || math.IsInf(value, 0) || value != math.Trunc(value) {
		return false
	}
	return value >= -float64(maxSafeInteger) && value <= float64(maxSafeInteger)
}

const maxSafeInteger int64 = 9007199254740991

func jsTrim(value string) string {
	return strings.TrimFunc(value, func(r rune) bool {
		switch r {
		case '\u0009', '\u000A', '\u000B', '\u000C', '\u000D', '\uFEFF', '\u2028', '\u2029':
			return true
		default:
			return unicode.Is(unicode.Zs, r)
		}
	})
}
