package cat

import (
	"context"
	"github.com/Station-Manager/types"
	"strings"
	"time"
)

func (s *Service) serialPortListener(shutdown <-chan struct{}) {
	readTicker := time.NewTicker(s.config.CatConfig.RateLimiterInterval * time.Millisecond)
	defer readTicker.Stop()

	var currentCancel context.CancelFunc

	for {
		select {
		case <-shutdown:
			if currentCancel != nil {
				currentCancel()
				currentCancel = nil
			}
			return
		case <-readTicker.C:
			s.LoggerService.DebugWith().Msg("Serial port listener tick")
			if currentCancel != nil {
				currentCancel()
				currentCancel = nil
			}

			ctx, cancel := context.WithTimeout(context.Background(), s.config.SerialConfig.ReadTimeout*time.Millisecond)
			currentCancel = cancel

			lineBytes, err := s.serialPort.ReadResponseBytes(ctx)

			currentCancel = nil
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
