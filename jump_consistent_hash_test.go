package dynproxy

import (
	"math"
	"math/rand"
	"testing"
)

func BenchmarkJumpHash(b *testing.B) {
	const buckets = 20
	key := rand.Int63n(math.MaxInt64)
	hash := JumpHash(uint64(key), buckets)
	if hash < 0 || hash > buckets {
		b.Fatalf("Hash: %d", hash)
	}
}

func TestJumpHash(t *testing.T) {
	const buckets = 20
	for i := 0; i < 100000000; i++ {
		key := rand.Int63n(math.MaxInt64)
		hash := JumpHash(uint64(key), buckets)
		if hash < 0 || hash > buckets {
			t.Fatalf("Hash: %d", hash)
		}
	}
}

func TestJumpHashDistribution(t *testing.T) {
	const buckets = 10
	counter0 := 0
	counter1 := 0
	counter2 := 0
	counter3 := 0
	counter4 := 0
	counter5 := 0
	counter6 := 0
	counter7 := 0
	counter8 := 0
	counter9 := 0
	for i := 0; i < 100000000; i++ {
		key := rand.Int63n(math.MaxInt64)
		hash := JumpHash(uint64(key), buckets)
		if hash < 0 || hash > buckets {
			t.Fatalf("Hash: %d", hash)
		}
		switch hash {
		case 0:
			counter0++
		case 1:
			counter1++
		case 2:
			counter2++
		case 3:
			counter3++
		case 4:
			counter4++
		case 5:
			counter5++
		case 6:
			counter6++
		case 7:
			counter7++
		case 8:
			counter8++
		case 9:
			counter9++
		}
	}
	t.Logf("0: %d", counter0)
	t.Logf("1: %d", counter1)
	t.Logf("2: %d", counter2)
	t.Logf("3: %d", counter3)
	t.Logf("4: %d", counter4)
	t.Logf("5: %d", counter5)
	t.Logf("6: %d", counter6)
	t.Logf("7: %d", counter7)
	t.Logf("8: %d", counter8)
	t.Logf("9: %d", counter9)

	t.Logf("total: %d", counter0+counter1+counter2+counter3+counter4+counter5+counter6+counter7+counter8+counter9)
}
