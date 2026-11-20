package main

import "testing"

func TestExampleMazesValid(t *testing.T) {
	grids, err := buildMazes()
	if err != nil {
		t.Fatal(err)
	}
	for idx, g := range grids {
		n := 0
		for _, row := range g {
			for _, c := range row {
				if c == Dot || c == Power {
					n++
				}
			}
		}
		t.Logf("maze %d: %d dots+pellets", idx, n)
	}
}
