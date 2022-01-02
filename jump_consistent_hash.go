package dynproxy

const MagicNumber = uint64(2862933555777941757)

func JumpHash(key uint64, numBuckets int) int {
	var bucket int64 = -1 // bucket number before the previous jump
	var jump int64 = 0    // bucket number before the current jump
	for jump < int64(numBuckets) {
		bucket = jump
		key = key*MagicNumber + 1
		jump = int64(float64(bucket+1) * (float64(int64(1)<<31) / float64((key>>33)+1)))
	}
	return int(bucket)
}
