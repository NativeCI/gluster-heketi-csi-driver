package util

// Common allocation units
const (
	Bytes int64 = 1
	KB    int64 = 1000 * Bytes
	MB    int64 = 1000 * KB
	GB    int64 = 1000 * MB
	TB    int64 = 1000 * GB

	minReplicaCount = 1
	maxReplicaCount = 10
)

func FromBytesToGb(bytes int64) int {
	return int(bytes / GB)
}

func FromGbToBytes(gb int64) int64 {
	return int64(gb * GB)
}
