package profilex

import (
	"os"

	"github.com/pkg/profile"
)

type noop struct{}

// Stop is a noop.
func (p *noop) Stop() {}

// Profile parses the PROFILING environment variable and executes the proper profiling task.
func Profile() interface {
	Stop()
} {
	switch os.Getenv("PROFILING") {
	case "cpu":
		return profile.Start(profile.CPUProfile)
	case "mem":
		return profile.Start(profile.MemProfile)
	case "mutex":
		return profile.Start(profile.MutexProfile)
	case "block":
		return profile.Start(profile.BlockProfile)
	}
	return new(noop)
}

// HelpMessage returns a string explaining how profiling works.
func HelpMessage() string {
	return `- PROFILING: Set "PROFILING=cpu" to enable cpu profiling and "PROFILING=memory" to enable memory profiling.
	It is not possible to do both at the same time. DProfiling is disabled per default.

	Example: PROFILING=cpu`
}
