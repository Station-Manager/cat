package cat

import (
	"context"
	stderr "errors"
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

	readTimeout := s.config.CatConfig.ListenerReadTimeoutMS
	if readTimeout <= 0 {
		readTimeout = defaultListenerReadTimeoutMS
	}
	readTimeout *= time.Millisecond

	for {
		select {
		case <-shutdown:
			return
		case <-readTicker.C:
			//s.LoggerService.DebugWith().Msg("Serial port listener tick")

			ctx, cancel := context.WithTimeout(context.Background(), readTimeout)

			lineBytes, err := s.serialPort.ReadResponseBytes(ctx)
			cancel()

			if err != nil {
				if stderr.Is(err, context.DeadlineExceeded) {
					continue
				}
				s.LoggerService.ErrorWith().Err(err).Msg("serial read failed")
				continue
			}

			if len(lineBytes) == 0 {
				continue
			}

			//			s.LoggerService.DebugWith().Msgf("Read response: %s", lineBytes)

			state, ok := s.lookupCatState(lineBytes)
			if !ok {
				continue
			}

			// We are interested in this state, so send it for processing
			select {
			case <-shutdown:
				return
			case s.processingChannel <- state:
				// delivered to the processing goroutine
			default:
				// Drop to avoid blocking/backpressure
			}
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
			// Store the line minus the matched prefix (as a string) in Data field.
			if len(line) >= l {
				st.Data = string(line)
			} else {
				st.Data = ""
			}
			return st, true
		}
	}

	return types.CatState{}, false
}
