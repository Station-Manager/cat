package cat

import (
	"github.com/Station-Manager/config"
	"github.com/Station-Manager/errors"
	"github.com/Station-Manager/logging"
	"github.com/Station-Manager/types"
	"sync"
	"sync/atomic"
)

const (
	ServiceName = types.CatServiceName
)

type Service struct {
	ConfigService *config.Service  `di.inject:"configservice"`
	LoggerService *logging.Service `di.inject:"loggingservice"`
	config        *types.RigConfig

	initialized atomic.Bool

	initOnce sync.Once
}

func (s *Service) Initialize() error {
	const op errors.Op = "cat.Service.Initialize"

	var initErr error
	s.initOnce.Do(func() {
		if s.ConfigService == nil {
			initErr = errors.New(op).Msg(errMsgNilConfigService)
			return
		}

		if s.LoggerService == nil {
			initErr = errors.New(op).Msg(errMsgNilLoggerService)
			return
		}

		cfg, err := s.getRigConfig()
		if err != nil {
			initErr = err
			return
		}

		if err = validateConfig(cfg); err != nil {
			initErr = errors.New(op).Err(err)
			return
		}

		s.initialized.Store(true)
	})

	return initErr
}
