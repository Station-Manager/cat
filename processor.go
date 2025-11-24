package cat

func (s *Service) lineProcessor(shutdown <-chan struct{}) {
	for {
		select {
		case <-shutdown:
			return
		case state := <-s.processingChannel:
			s.LoggerService.DebugWith().Str("line", state.Data).Msg("Processing line")
		}
	}
}
