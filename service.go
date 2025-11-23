package cat

import (
	"github.com/Station-Manager/config"
	"github.com/Station-Manager/errors"
	"github.com/Station-Manager/logging"
	"github.com/Station-Manager/serial"
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

	serialPort *serial.Port

	supportedCatStates map[string]types.CatState

	shutdownChannel chan bool

	initialized atomic.Bool
	started     bool
	stopped     bool

	initOnce sync.Once
	mu       sync.Mutex
	wg       *sync.WaitGroup
}

// Initialize ensures the service is properly set up by initializing required components and loading configurations.
// It is safe to call multiple times. The IOCDI container will ensure this method is called.
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
			initErr = err
			return
		}

		s.config = cfg

		s.initializeStateSet()

		s.wg = &sync.WaitGroup{}

		s.shutdownChannel = make(chan bool)

		s.initialized.Store(true)
	})

	return initErr
}

func (s *Service) Start() error {
	const op errors.Op = "cat.Service.Start"
	if !s.initialized.Load() {
		return errors.New(op).Msg("Service not initialized.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return errors.New(op).Msg("Service already started.")
	}

	if err := s.initializeSerialPort(); err != nil {
		return errors.New(op).Err(err).Msg("Failed to initialize serial port.")
	}

	s.launchWorkerThread(s.serialPortListener, "serialPortListener")
	s.launchWorkerThread(s.serialPortSender, "serialPortSender")

	s.started = true

	return nil
}

func (s *Service) Stop() error {
	const op errors.Op = "cat.Service.Stop"
	if !s.initialized.Load() {
		return errors.New(op).Msg("Service not initialized.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return errors.New(op).Msg("Service not started.")
	}

	if s.shutdownChannel != nil {
		close(s.shutdownChannel)
		s.shutdownChannel = nil
	}

	if s.wg != nil {
		s.wg.Wait()
	}

	if err := s.serialPort.Close(); err != nil {
		return errors.New(op).Msgf("Failed to close serial port: %v", err)
	}
	s.serialPort = nil
	s.started = false

	return nil
}
