package cat

import "time"

func (s *Service) serialPortListener(shutdown <-chan struct{}) {
	readTicker := time.NewTicker(s.config.CatConfig.RateLimiterInterval * time.Millisecond)
	defer readTicker.Stop()

	for {
		select {
		case <-shutdown:
			return
		case <-readTicker.C:
			s.LoggerService.DebugWith().Msg("Serial port listener tick")
		}
	}
}
