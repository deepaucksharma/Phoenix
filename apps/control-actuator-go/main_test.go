package main

import "testing"

func TestCalculateMode(t *testing.T) {
	cl := &ControlLoop{
		conservativeMax:  100,
		aggressiveMin:    200,
		hysteresisFactor: 0.1,
		currentMode:      Balanced,
	}

	if mode := cl.calculateMode(80, -1); mode != Conservative {
		t.Errorf("expected %s, got %s", Conservative, mode)
	}

	if mode := cl.calculateMode(220, 1); mode != Aggressive {
		t.Errorf("expected %s, got %s", Aggressive, mode)
	}

	if mode := cl.calculateMode(150, 0); mode != Balanced {
		t.Errorf("expected %s, got %s", Balanced, mode)
	}

	cl.currentMode = Aggressive
	if mode := cl.calculateMode(190, -1); mode != Balanced {
		t.Errorf("expected %s due to hysteresis, got %s", Balanced, mode)
	}

	cl.currentMode = Conservative
	if mode := cl.calculateMode(105, -1); mode != Conservative {
		t.Errorf("expected %s to remain due to hysteresis, got %s", Conservative, mode)
	}
}
