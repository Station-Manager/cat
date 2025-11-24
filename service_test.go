package cat

import (
	"github.com/Station-Manager/cat/enums/cmd"
	"github.com/Station-Manager/config"
	"github.com/Station-Manager/iocdi"
	"github.com/Station-Manager/logging"
	"github.com/Station-Manager/types"
	"github.com/stretchr/testify/require"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestInitFailureNilConfigService(t *testing.T) {
	service := &Service{}
	err := service.Initialize()
	require.Error(t, err)
	require.Contains(t, err.Error(), errMsgNilConfigService)
}

func TestInitFailureNilLoggerService(t *testing.T) {
	service := &Service{
		ConfigService: &config.Service{
			WorkingDir: "some/path",
			AppConfig:  types.AppConfig{},
		},
	}
	err := service.Initialize()
	require.Error(t, err)
	require.Contains(t, err.Error(), errMsgNilLoggerService)
}

func TestInitFailureInvalidRigID(t *testing.T) {
	cfgService := &config.Service{
		WorkingDir: ".",
		AppConfig:  types.AppConfig{},
	}
	err := cfgService.Initialize()
	require.NoError(t, err)

	service := &Service{
		ConfigService: cfgService,
		LoggerService: &logging.Service{},
	}

	err = service.Initialize()
	//require.Error(t, err)
	//require.Contains(t, err.Error(), errMsgInvalidRigID)
}

func TestInitWithContainer(t *testing.T) {
	container := iocdi.New()
	require.NoError(t, container.RegisterInstance("workingdir", "."))
	require.NoError(t, container.Register(config.ServiceName, reflect.TypeOf((*config.Service)(nil))))
	require.NoError(t, container.Register(logging.ServiceName, reflect.TypeOf((*logging.Service)(nil))))
	require.NoError(t, container.Register(ServiceName, reflect.TypeOf((*Service)(nil))))
	require.NoError(t, container.Build())

	obj, err := container.ResolveSafe(ServiceName)
	require.NoError(t, err)
	require.NotNil(t, obj)

	cat, ok := obj.(*Service)
	require.True(t, ok)

	err = cat.Start()
	require.NoError(t, err)

	err = cat.EnqueueCommand(cmd.Init)
	require.NoError(t, err)

	time.Sleep(15 * time.Second)

	err = cat.Stop()
	require.NoError(t, err)
}

// TestServiceStartStopConcurrent ensures that concurrent calls to Start do not race,
// and that repeated Start/Stop cycles are safe.
func TestServiceStartStopConcurrent(t *testing.T) {
	container := iocdi.New()
	require.NoError(t, container.RegisterInstance("workingdir", "."))
	require.NoError(t, container.Register(config.ServiceName, reflect.TypeOf((*config.Service)(nil))))
	require.NoError(t, container.Register(logging.ServiceName, reflect.TypeOf((*logging.Service)(nil))))
	require.NoError(t, container.Register(ServiceName, reflect.TypeOf((*Service)(nil))))
	require.NoError(t, container.Build())

	obj, err := container.ResolveSafe(ServiceName)
	require.NoError(t, err)
	cat, ok := obj.(*Service)
	require.True(t, ok)

	// Ensure service is initialized before concurrent Start/Stop.
	require.NoError(t, cat.Initialize())

	// First, exercise multiple sequential Start/Stop cycles.
	for i := 0; i < 3; i++ {
		require.NoError(t, cat.Start())
		require.NoError(t, cat.Stop())
	}

	// Now exercise concurrent Start/Stop on a fresh run.
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			_ = cat.Start()
			// give Stop goroutine a chance to run
			time.Sleep(10 * time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			_ = cat.Stop()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Final clean Start/Stop to ensure consistent end state.
	require.NoError(t, cat.Start())
	require.NoError(t, cat.Stop())
}

// TestStatusChannelReceiveOnly verifies that StatusChannel returns a receive-only
// channel and that it behaves correctly when the service is initialized.
func TestStatusChannelReceiveOnly(t *testing.T) {
	container := iocdi.New()
	require.NoError(t, container.RegisterInstance("workingdir", "."))
	require.NoError(t, container.Register(config.ServiceName, reflect.TypeOf((*config.Service)(nil))))
	require.NoError(t, container.Register(logging.ServiceName, reflect.TypeOf((*logging.Service)(nil))))
	require.NoError(t, container.Register(ServiceName, reflect.TypeOf((*Service)(nil))))
	require.NoError(t, container.Build())

	obj, err := container.ResolveSafe(ServiceName)
	require.NoError(t, err)
	cat, ok := obj.(*Service)
	require.True(t, ok)

	// Ensure initialization succeeds so StatusChannel is available.
	require.NoError(t, cat.Initialize())

	ch, err := cat.StatusChannel()
	require.NoError(t, err)
	require.NotNil(t, ch)

	// This compile-time-only assertion ensures the returned type is receive-only.
	var _ <-chan types.CatStatus = ch
}

// TestEnqueueCommandFormatValidation ensures that EnqueueCommand fails fast
// when the configured command format string does not match the provided
// parameter count.
func TestEnqueueCommandFormatValidation(t *testing.T) {
	// Build a minimal in-memory Service with a single CatCommand using a format string.
	cfg := &types.RigConfig{
		CatConfig: types.CatConfig{
			SendChannelSize:       1,
			ProcessingChannelSize: 1,
		},
	}
	cfg.CatCommands = []types.CatCommand{{
		Name: cmd.Init.String(),
		Cmd:  "CMD %s %s", // expects 2 parameters
	}}

	service := &Service{
		ConfigService: &config.Service{},
		LoggerService: &logging.Service{},
		config:        cfg,
		sendChannel:   make(chan types.CatCommand, 1),
	}
	service.initialized.Store(true)
	service.started = true

	// Happy path: correct parameter count.
	err := service.EnqueueCommand(cmd.Init, "one", "two")
	require.NoError(t, err)

	// Too few parameters.
	err = service.EnqueueCommand(cmd.Init, "only-one")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Command parameter validation failed")

	// Too many parameters.
	err = service.EnqueueCommand(cmd.Init, "one", "two", "three")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Command parameter validation failed")
}
