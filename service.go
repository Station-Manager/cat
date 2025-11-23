package cat

import (
	"github.com/Station-Manager/config"
	"github.com/Station-Manager/logging"
	"github.com/Station-Manager/types"
)

const (
	ServiceName = types.CatServiceName
)

type Service struct {
	ConfigService *config.Service  `di.inject:"configservice"`
	LoggerService *logging.Service `di.inject:"loggingservice"`
	config        *types.RigConfig
}
