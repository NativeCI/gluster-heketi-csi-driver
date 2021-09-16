package util

// Common allocation units
const (
	KB int64 = 1000
	MB int64 = 1000 * KB
	GB int64 = 1000 * MB
	TB int64 = 1000 * GB

	minReplicaCount = 1
	maxReplicaCount = 10
)

func FromBytesToGb(bytes int64) int {
	return int(bytes / 1024 / 1024 / 1024 / 1024)
}
