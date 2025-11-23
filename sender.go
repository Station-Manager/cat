package cat

func (s *Service) serialPortSender() {
	for {
		select {
		case <-s.shutdownChannel:
			return
		}
	}
}
