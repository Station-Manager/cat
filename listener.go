package cat

import (
	"context"
	"github.com/Station-Manager/types"
	"strings"
	"time"
)

const (
	defaultListenerReadTimeoutMS = 200
)

// serialPortListener listens for and processes data from a serial port at a set interval until a shutdown signal is received.
func (s *Service) serialPortListener(shutdown <-chan struct{}) {
	readTicker := time.NewTicker(s.config.CatConfig.ListenerRateLimiterInterval * time.Millisecond)
	defer readTicker.Stop()

	for {
		select {
		case <-shutdown:
			return
		case <-readTicker.C:
			s.LoggerService.DebugWith().Msg("Serial port listener tick")

			// Prefer the CAT-level listener timeout when configured; otherwise fall
			// back to the serial driver's read timeout. This keeps transport
			// concerns (SerialConfig) separate from CAT polling behavior.
			readTimeout := s.config.CatConfig.ListenerReadTimeoutMS
			if readTimeout <= 0 {
				readTimeout = defaultListenerReadTimeoutMS
			}

			ctx, cancel := context.WithTimeout(context.Background(), readTimeout*time.Millisecond)

			lineBytes, err := s.serialPort.ReadResponseBytes(ctx)
			cancel()

			if err != nil {
				s.LoggerService.ErrorWith().Err(err).Msg("serial read failed")
				continue
			}

			state, ok := s.lookupCatState(lineBytes)
			if !ok {
				continue
			}
			s.LoggerService.DebugWith().Msgf("Found cat state: %s", state.Prefix)
		}
	}
}

// lookupCatState attempts to find a CatState based on the byte slice prefix, returning the state and a success indicator.
func (s *Service) lookupCatState(line []byte) (types.CatState, bool) {
	const minPrefix = 2

	if len(line) < minPrefix {
		return types.CatState{}, false
	}

	// determine how many bytes to inspect
	maxLen := s.maxCatPrefixLen
	if maxLen > len(line) {
		maxLen = len(line)
	}
	if maxLen < minPrefix {
		maxLen = minPrefix
	}

	// take the slice once, uppercase it for consistent lookup
	prefixSlice := strings.ToUpper(string(line[:maxLen]))

	// try longest first to match multi-char prefixes (3..8) before 2-char ones
	for l := maxLen; l >= minPrefix; l-- {
		key := strings.TrimSpace(prefixSlice[:l])
		if key == "" {
			continue
		}
		if st, ok := s.supportedCatStates[key]; ok {
			return st, true
		}
	}

	return types.CatState{}, false
}
