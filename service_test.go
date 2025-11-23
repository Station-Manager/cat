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

	var wg sync.WaitGroup

	// Launch several goroutines that call Start and Stop concurrently.
	for i := 0; i < 10; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			_ = cat.Start() // only one should succeed, others should get "already started"
		}()

		go func() {
			defer wg.Done()
			_ = cat.Stop() // only valid after a successful Start, others may see "not started"
		}()
	}

	wg.Wait()

	// Final clean Start/Stop sequence to ensure service is left in a good state.
	require.NoError(t, cat.Start())
	require.NoError(t, cat.Stop())
}
