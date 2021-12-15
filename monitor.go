package dynproxy

import (
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

func GetOSConfig() {
	noRLimit := &unix.Rlimit{}
	err := unix.Getrlimit(unix.RLIMIT_NOFILE, noRLimit)
	if err != nil {
		log.Error().Msgf("error occur while getting OS limit of open files: %+v", err)
	}
	err = unix.Setrlimit(unix.RLIMIT_NOFILE, &unix.Rlimit{
		Cur: 4096,
		Max: 100000,
	})
	if err != nil {
		log.Error().Msgf("error occur while getting OS limit of open files: %+v", err)
	}
}
