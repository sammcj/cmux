// internal/workspace/race_race_test.go
//go:build race

package workspace

// raceDetectorEnabled returns true when compiled with race detector
func raceDetectorEnabled() bool {
	return true
}
