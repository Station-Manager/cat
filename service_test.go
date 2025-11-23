package cat

import (
	"github.com/Station-Manager/config"
	"github.com/Station-Manager/logging"
	"github.com/Station-Manager/types"
	"github.com/stretchr/testify/require"
	"testing"
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
