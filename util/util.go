// Copyright 2015 Apcera Inc. All rights reserved.

package util

import (
	"math/rand"
	"time"
)

// Random generates a random number in between min and max
// If min equals max then min is returned. If max is less than min
// then the function panics.
func Random(min, max int) int {
	if min == max {
		return min
	}
	if max < min {
		panic("max cannot be less than min")
	}
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Intn(max-min+1) + min
}
