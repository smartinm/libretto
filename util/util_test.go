// Copyright 2016 Apcera Inc. All rights reserved.

package util

import "testing"

const (
	maxSamplingSize = 100000
)

// TestRandomSingle makes sure an equal min/max always returns that value.
func TestRandomSingle(t *testing.T) {
	testMinMax := []int{1, 50, 100, 642}

	for _, val := range testMinMax {
		min := val
		max := val

		m := sampleRandom(min, max, 10)

		_, ok := m[min]
		if !ok {
			t.Fatalf("Failed to contain expected value in sampling: %d\n", min)
		}

		count := len(m)
		if count != 1 {
			t.Fatalf("Found value out of bounds.\n")
		}
	}
}

// TestRandomCouple makes sure min and max with a difference of only 1, returns both values.
func TestRandomCouple(t *testing.T) {
	testMinMax := []int{1, 50, 100, 642}

	for _, val := range testMinMax {
		min := val
		max := val + 1

		m := sampleRandom(min, max, 1000)

		_, ok := m[min]
		if !ok {
			t.Fatalf("Failed to contain min value in sampling: %d\n", min)
		}

		_, ok = m[max]
		if !ok {
			t.Fatalf("Failed to contain max value in sampling: %d\n", max)
		}

		count := len(m)
		if count != 2 {
			t.Fatalf("Found value out of bounds.\n")
		}
	}
}

func TestRandomSet(t *testing.T) {
	testMinMax := [][]int{
		{1, 5},
		{50, 58},
		{100, 104},
		{643, 650},
	}

	testValues := [][]int{
		{1, 2, 3, 4, 5},
		{50, 51, 52, 53, 54, 58},
		{100, 101, 102, 103, 104},
		{643, 644, 645, 646, 647, 648, 649, 650},
	}

	for i, val := range testMinMax {
		min := val[0]
		max := val[1]

		m := sampleRandom(min, max, maxSamplingSize)

		for _, val := range testValues[i] {
			_, ok := m[val]
			if !ok {
				t.Fatalf("Failed to contain value in sampling: %d\n", val)
			}
		}

		count := len(m)
		if count != max-min+1 {
			t.Fatalf("Found value out of bounds.\n")
		}
	}
}

func sampleRandom(min int, max int, s int) map[int]interface{} {
	m := make(map[int]interface{})

	for i := 1; i <= s; i++ {
		m[Random(min, max)] = nil
	}

	return m
}
