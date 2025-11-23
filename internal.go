package cat

import (
	"github.com/Station-Manager/errors"
	"github.com/Station-Manager/types"
)

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
