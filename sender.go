package cat

func (s *Service) serialPortSender(shutdown <-chan struct{}) {
	for {
		select {
		case <-shutdown:
			return
		}
	}
}
