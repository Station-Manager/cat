package cat

import (
	"github.com/Station-Manager/errors"
	"github.com/Station-Manager/serial"
	"github.com/Station-Manager/types"
	"strings"
)

// getRigConfig retrieves the default rig configuration based on the default rig ID from the required configurations.
// It returns an error if no valid default rig ID is set or if fetching the rig configuration fails.
func (s *Service) getRigConfig() (*types.RigConfig, error) {
	const op errors.Op = "cat.Service.getRigConfig"
	rigConfigs, err := s.ConfigService.RequiredConfigs()
	if err != nil {
		return nil, errors.New(op).Err(err)
	}

	defaultRigID := rigConfigs.DefaultRigID
	if defaultRigID < 1 {
		return nil, errors.New(op).Msg(errMsgInvalidRigID)
	}

	cfg, err := s.ConfigService.RigConfigByID(defaultRigID)
	if err != nil {
		return nil, errors.New(op).Err(err)
	}

	return &cfg, nil
}

// initializeSerialPort initializes the serial port using the provided configuration in the Service instance.
// It returns an error if the serial port cannot be opened.
func (s *Service) initializeSerialPort() error {
	const op errors.Op = "cat.Service.initializeSerialPort"

	var err error
	s.serialPort, err = serial.Open(s.config.SerialConfig)
	if err != nil {
		return errors.New(op).Err(err)
	}

	return nil
}

// initializeStateSet initializes the supportedCatStates map based on the configured CatState values in the service.
func (s *Service) initializeStateSet() {
	s.supportedCatStates = make(map[string]types.CatState, len(s.config.CatStates))

	for _, state := range s.config.CatStates {
		key := strings.ToUpper(strings.TrimSpace(state.Prefix))
		if key == "" {
			// TODO: probably should log this?
			continue
		}
		s.supportedCatStates[key] = state
	}
}

// launchWorkerThread starts a new goroutine for the given worker function and manages its lifecycle using a wait group.
func (s *Service) launchWorkerThread(run *runState, workerFunc func(<-chan struct{}), workerName string) {
	run.wg.Add(1)
	go func() {
		defer run.wg.Done()
		s.LoggerService.InfoWith().Str("worker", workerName).Msg("CAT starting")
		workerFunc(run.shutdownChannel)
		s.LoggerService.InfoWith().Str("worker", workerName).Msg("CAT stopped")
	}()
}
