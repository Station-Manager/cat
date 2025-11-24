package cat

import (
	"fmt"
	"github.com/Station-Manager/cat/enums/cmd"
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
	// defaultListenerIntervalMS is used when the configured ListenerRateLimiterInterval
	// is zero or negative, to avoid creating a ticker with a non-positive duration.
	defaultListenerIntervalMS = 250
)

type runState struct {
	shutdownChannel chan struct{}
	wg              sync.WaitGroup
}

type Service struct {
	ConfigService *config.Service  `di.inject:"configservice"`
	LoggerService *logging.Service `di.inject:"loggingservice"`
	config        *types.RigConfig

	serialPort *serial.Port

	supportedCatStates map[string]types.CatState
	maxCatPrefixLen    int

	initialized atomic.Bool
	started     bool

	initOnce sync.Once
	mu       sync.Mutex

	currentRun *runState

	statusChannel chan types.CatStatus
	sendChannel   chan types.CatCommand
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

		// Ensure sensible defaults for CAT timing configuration to avoid panics
		// when creating tickers or timeouts with non-positive durations.
		if cfg.CatConfig.ListenerRateLimiterInterval <= 0 {
			cfg.CatConfig.ListenerRateLimiterInterval = defaultListenerIntervalMS
		}
		if cfg.CatConfig.ListenerReadTimeoutMS <= 0 {
			cfg.CatConfig.ListenerReadTimeoutMS = cfg.SerialConfig.ReadTimeoutMS
		}

		s.config = cfg

		s.initializeStateSet()

		// This channel is non-blocking and buffered to avoid deadlocks. Leaving it a 1 ensures that
		// the status stream is “latest-wins” so that the caller (the frontend) should not lag behind.
		s.statusChannel = make(chan types.CatStatus, 1)
		s.sendChannel = make(chan types.CatCommand, s.config.CatConfig.SendChannelSize)

		s.initialized.Store(true)
	})

	return initErr
}

// Start initializes and starts the service if it has been properly configured and is not already running.
func (s *Service) Start() error {
	const op errors.Op = "cat.Service.Start"
	if !s.initialized.Load() {
		return errors.New(op).Msg(errMsgServiceNotInit)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return errors.New(op).Msg("Service already started.")
	}

	if err := s.initializeSerialPort(); err != nil {
		return errors.New(op).Err(err).Msg("Failed to initialize serial port.")
	}

	run := &runState{
		shutdownChannel: make(chan struct{}),
	}
	s.currentRun = run

	s.launchWorkerThread(run, s.serialPortListener, "serialPortListener")
	s.launchWorkerThread(run, s.serialPortSender, "serialPortSender")

	s.started = true

	return nil
}

// Stop safely stops the service by shutting down active processes, releasing resources, and closing the serial port.
func (s *Service) Stop() error {
	const op errors.Op = "cat.Service.Stop"
	if !s.initialized.Load() {
		return errors.New(op).Msg(errMsgServiceNotInit)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return errors.New(op).Msg(errMsgServiceNotStarted)
	}

	run := s.currentRun
	if run != nil && run.shutdownChannel != nil {
		close(run.shutdownChannel)
	}
	// NOTE: we do not close any of the other channels here, as they may be in use by other goroutines
	// which would panic on 'send' if the channel were closed. All goroutines exit via shutdownChannel,
	// so these channels will eventually become unreachable and be garbage-collected.

	if run != nil {
		run.wg.Wait()
	}

	if s.serialPort != nil {
		if err := s.serialPort.Close(); err != nil {
			return errors.New(op).Msgf("Failed to close serial port: %v", err)
		}
		s.serialPort = nil
	}

	s.currentRun = nil
	s.started = false

	return nil
}

// StatusChannel returns a channel for monitoring cat status changes or an error if the service is uninitialized or closed.
func (s *Service) StatusChannel() (chan types.CatStatus, error) {
	const op errors.Op = "cat.Service.StatusChannel"
	if !s.initialized.Load() {
		return nil, errors.New(op).Msg(errMsgServiceNotInit)
	}

	if s.statusChannel == nil {
		return nil, errors.New(op).Msg("Status channel is closed.")
	}

	return s.statusChannel, nil
}

func (s *Service) EnqueueCommand(cmdName cmd.CatCmdName, params ...string) error {
	const op errors.Op = "cat.Service.EnqueueCommand"
	if !s.initialized.Load() {
		return errors.New(op).Msg(errMsgServiceNotInit)
	}

	if !s.started {
		return errors.New(op).Msg(errMsgServiceNotStarted)
	}

	catCmd, err := s.commandLookup(cmdName)
	if err != nil {
		return errors.New(op).Msgf("Command lookup failed: %v", err)
	}

	paramsInterface := make([]interface{}, len(params))
	for i, v := range params {
		paramsInterface[i] = v
	}

	pCmd := &catCmd
	pCmd.Cmd = fmt.Sprintf(catCmd.Cmd, paramsInterface...)
	catCmd = *pCmd

	//TODO: validate/sanitize command?

	if s.sendChannel != nil {
		select {
		case s.sendChannel <- catCmd:
			return nil
		default:
			return errors.New(op).Msg("Send channel is full.")
		}
	}
	return errors.New(op).Msg("Send channel is closed.")
}
