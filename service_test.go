package cat

import (
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

	time.Sleep(5 * time.Second)

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
