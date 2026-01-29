// internal/workspace/race_test.go
//go:build !race

package workspace

// raceDetectorEnabled returns false when not compiled with race detector
func raceDetectorEnabled() bool {
	return false
}
