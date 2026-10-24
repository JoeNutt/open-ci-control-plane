package fixture

import "testing"

func TestAdmitsAnOwner(t *testing.T) {
	t.Log("owner")
}

func helper(t *testing.T) {}

type suite struct{}

func (suite) TestMethodIsNotATest(t *testing.T) {}

func TestTakesTwoParameters(t *testing.T, extra int) {}

func BenchmarkIsNotATest(b *testing.B) {}

func TestTakesABenchmark(b *testing.B) {}

func TestRefusesAStranger(t *testing.T) {
	t.Log("stranger")
	t.Log("refused")
}
