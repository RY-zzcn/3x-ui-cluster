package service

import (
	_ "embed"
	"encoding/json"

	"github.com/mhsanaei/3x-ui/v2/util/common"
	"github.com/mhsanaei/3x-ui/v2/xray"
)

// XraySettingService provides business logic for Xray configuration management.
// It handles validation and storage of Xray template configurations per slave.
type XraySettingService struct {
	SlaveSettingService
}

// SaveXraySettingForSlave validates and saves xrayTemplateConfig for a specific slave.
func (s *XraySettingService) SaveXraySettingForSlave(slaveId int, newXraySettings string) error {
	if err := s.CheckXrayConfig(newXraySettings); err != nil {
		return err
	}
	return s.SlaveSettingService.SaveXrayConfigForSlave(slaveId, newXraySettings)
}

// GetXraySettingForSlave retrieves the xrayTemplateConfig for a specific slave.
func (s *XraySettingService) GetXraySettingForSlave(slaveId int) (string, error) {
	return s.SlaveSettingService.GetXrayConfigForSlave(slaveId)
}

func (s *XraySettingService) CheckXrayConfig(XrayTemplateConfig string) error {
	xrayConfig := &xray.Config{}
	err := json.Unmarshal([]byte(XrayTemplateConfig), xrayConfig)
	if err != nil {
		return common.NewError("xray template config invalid:", err)
	}
	return nil
}
