package clock

import (
	"reflect"
	"testing"
)

func TestBasicInit(t *testing.T) {
	n := NewClock("a")

	n.Tick()

	na, isFounda := n.FindTicks("a")

	if !isFounda {
		t.Fatalf("Failed on finding ticks: %s", n)
	}

	if na != 1 {
		t.Fatalf("Tick value did not increment: %s", n)
	}

}

func TestCopy(t *testing.T) {
	clock := ClockMap{
		"a": 4,
		"b": 1,
		"c": 3,
		"d": 2,
	}

	n := NewClock("a")
	n.Merge(clock)

	ao, _ := n.FindTicks("a")
	bo, _ := n.FindTicks("b")
	co, _ := n.FindTicks("c")
	do, _ := n.FindTicks("d")

	if clock["a"] != ao || clock["b"] != bo || clock["c"] != co || clock["d"] != do {
		t.Fatalf("Copy not the same as the original new = %v , old = %v ", n.GetClock(), clock)
	} else if !reflect.DeepEqual(n.GetClock(), clock) {
		t.Fatalf("Copy not the same as the original new = %v , old = %v ", n.GetClock(), clock)
	}
}

func TestAndMerge(t *testing.T) {
	na := NewClock("a")
	nb := NewClock("b")

	clockA := ClockMap{
		"a": 2,
		"b": 1,
		"c": 1,
	}
	na.Merge(clockA)

	clockB := ClockMap{
		"a": 1,
		"b": 3,
		"c": 1,
	}
	nb.Merge(clockB)
	nb.Merge(na.GetClock())

	an, _ := nb.FindTicks("a")
	bn, _ := nb.FindTicks("b")
	cn, _ := nb.FindTicks("c")

	if an != 2 || bn != 3 || cn != 1 {
		t.Fatalf("Merge not as expected")
	}
}
