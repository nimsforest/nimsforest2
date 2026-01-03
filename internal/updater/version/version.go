package version

import (
	"strconv"
	"strings"
)

// Compare compares two semantic version strings (e.g., "1.2.3" vs "1.2.4")
// Supports extended versioning with any number of parts (e.g., "1.2.7.6.7")
// Returns:
//
//	 1 if v1 > v2
//	 0 if v1 == v2
//	-1 if v1 < v2
func Compare(v1, v2 string) int {
	// Remove 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var num1, num2 int

		if i < len(parts1) {
			// Parse number, ignoring any non-numeric suffix (e.g., "1.2.3-beta")
			numStr := strings.Split(parts1[i], "-")[0]
			num1, _ = strconv.Atoi(numStr)
		}

		if i < len(parts2) {
			numStr := strings.Split(parts2[i], "-")[0]
			num2, _ = strconv.Atoi(numStr)
		}

		if num1 > num2 {
			return 1
		}
		if num1 < num2 {
			return -1
		}
	}

	return 0
}

// IsNewer returns true if newVersion is newer than currentVersion
func IsNewer(newVersion, currentVersion string) bool {
	return Compare(newVersion, currentVersion) > 0
}
