package service

import (
	"encoding/json"
	"fmt"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/logger"
	"github.com/mhsanaei/3x-ui/v2/xray"

	"gorm.io/gorm"
)

// OutboundService provides business logic for managing Xray outbound configurations.
// Outbound rules are stored directly in the xrayTemplateConfig JSON in the slave_settings table.
// Traffic stats are stored in the OutboundTraffics table.
type OutboundService struct {
	SlaveSettingService SlaveSettingService
}

// ===== Template-based Outbound Rule Management =====

// getTemplateOutbounds parses the xrayTemplateConfig for a slave and returns the outbounds array
func (s *OutboundService) getTemplateOutbounds(slaveId int) ([]map[string]interface{}, error) {
	templateJson, err := s.SlaveSettingService.GetXrayConfigForSlave(slaveId)
	if err != nil {
		return nil, fmt.Errorf("failed to get xray template config for slave %d: %v", slaveId, err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(templateJson), &config); err != nil {
		return nil, fmt.Errorf("failed to parse xray template config: %v", err)
	}

	outboundsRaw, ok := config["outbounds"]
	if !ok {
		return []map[string]interface{}{}, nil
	}

	outboundsArr, ok := outboundsRaw.([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	result := make([]map[string]interface{}, 0, len(outboundsArr))
	for _, item := range outboundsArr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result, nil
}

// saveTemplateOutbounds updates the outbounds array in xrayTemplateConfig for a slave and saves it
func (s *OutboundService) saveTemplateOutbounds(slaveId int, outbounds []map[string]interface{}) error {
	templateJson, err := s.SlaveSettingService.GetXrayConfigForSlave(slaveId)
	if err != nil {
		return fmt.Errorf("failed to get xray template config for slave %d: %v", slaveId, err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(templateJson), &config); err != nil {
		return fmt.Errorf("failed to parse xray template config: %v", err)
	}

	config["outbounds"] = outbounds

	newJson, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal xray template config: %v", err)
	}

	return s.SlaveSettingService.SaveXrayConfigForSlave(slaveId, string(newJson))
}

// GetOutbounds returns all outbound rules from the template config for a slave.
// Each outbound is returned with an "id" field set to its array index.
func (s *OutboundService) GetOutbounds(slaveId int) ([]map[string]interface{}, error) {
	outbounds, err := s.getTemplateOutbounds(slaveId)
	if err != nil {
		return nil, err
	}

	// Add pseudo-ID (array index) for frontend
	for i := range outbounds {
		outbounds[i]["id"] = i
	}
	return outbounds, nil
}

// AddOutbound adds a new outbound rule to the template config for a slave
func (s *OutboundService) AddOutbound(slaveId int, outbound map[string]interface{}) error {
	outbounds, err := s.getTemplateOutbounds(slaveId)
	if err != nil {
		return err
	}

	// Remove any frontend-generated id
	delete(outbound, "id")

	outbounds = append(outbounds, outbound)
	logger.Infof("Added outbound rule for slave %d, total outbounds: %d", slaveId, len(outbounds))
	return s.saveTemplateOutbounds(slaveId, outbounds)
}

// UpdateOutbound updates an outbound rule at the given index in the template config for a slave
func (s *OutboundService) UpdateOutbound(slaveId int, index int, outbound map[string]interface{}) error {
	outbounds, err := s.getTemplateOutbounds(slaveId)
	if err != nil {
		return err
	}

	if index < 0 || index >= len(outbounds) {
		return fmt.Errorf("outbound index %d out of range (total: %d)", index, len(outbounds))
	}

	// Remove any frontend-generated id
	delete(outbound, "id")

	outbounds[index] = outbound
	logger.Infof("Updated outbound rule at index %d for slave %d", index, slaveId)
	return s.saveTemplateOutbounds(slaveId, outbounds)
}

// DeleteOutbound removes an outbound rule at the given index from the template config for a slave
func (s *OutboundService) DeleteOutbound(slaveId int, index int) error {
	outbounds, err := s.getTemplateOutbounds(slaveId)
	if err != nil {
		return err
	}

	if index < 0 || index >= len(outbounds) {
		return fmt.Errorf("outbound index %d out of range (total: %d)", index, len(outbounds))
	}

	tag := ""
	if t, ok := outbounds[index]["tag"].(string); ok {
		tag = t
	}

	outbounds = append(outbounds[:index], outbounds[index+1:]...)
	logger.Infof("Deleted outbound rule at index %d (tag: %s) for slave %d, remaining: %d", index, tag, slaveId, len(outbounds))
	return s.saveTemplateOutbounds(slaveId, outbounds)
}

// ===== Traffic Stats (still uses OutboundTraffics table) =====

func (s *OutboundService) AddTraffic(traffics []*xray.Traffic, clientTraffics []*xray.ClientTraffic) (error, bool) {
	var err error
	db := database.GetDB()
	tx := db.Begin()

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	err = s.addOutboundTraffic(tx, traffics)
	if err != nil {
		return err, false
	}

	return nil, false
}

func (s *OutboundService) addOutboundTraffic(tx *gorm.DB, traffics []*xray.Traffic) error {
	if len(traffics) == 0 {
		return nil
	}

	var err error

	for _, traffic := range traffics {
		if traffic.IsOutbound {

			var outbound model.OutboundTraffics

			err = tx.Model(&model.OutboundTraffics{}).Where("tag = ?", traffic.Tag).
				FirstOrCreate(&outbound).Error
			if err != nil {
				return err
			}

			outbound.Tag = traffic.Tag
			outbound.Up = outbound.Up + traffic.Up
			outbound.Down = outbound.Down + traffic.Down
			outbound.Total = outbound.Up + outbound.Down

			err = tx.Save(&outbound).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *OutboundService) GetOutboundsTraffic() ([]*model.OutboundTraffics, error) {
	db := database.GetDB()
	var traffics []*model.OutboundTraffics

	err := db.Model(model.OutboundTraffics{}).Find(&traffics).Error
	if err != nil {
		logger.Warning("Error retrieving OutboundTraffics: ", err)
		return nil, err
	}

	return traffics, nil
}

func (s *OutboundService) GetOutboundsTrafficForSlave(slaveId int) ([]*model.OutboundTraffics, error) {
	db := database.GetDB()
	var traffics []*model.OutboundTraffics

	err := db.Model(model.OutboundTraffics{}).Where("slave_id = ?", slaveId).Find(&traffics).Error
	if err != nil {
		logger.Warning("Error retrieving OutboundTraffics for slave: ", err)
		return nil, err
	}

	return traffics, nil
}

func (s *OutboundService) ResetOutboundTraffic(tag string) error {
	db := database.GetDB()

	whereText := "tag "
	if tag == "-alltags-" {
		whereText += " <> ?"
	} else {
		whereText += " = ?"
	}

	result := db.Model(model.OutboundTraffics{}).
		Where(whereText, tag).
		Updates(map[string]any{"up": 0, "down": 0, "total": 0})

	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func (s *OutboundService) ResetOutboundTrafficForSlave(slaveId int, tag string) error {
	db := database.GetDB()

	query := db.Model(model.OutboundTraffics{}).Where("slave_id = ?", slaveId)

	if tag == "-alltags-" {
		// Reset all outbounds for this slave
		query = query.Where("tag <> ?", tag)
	} else {
		// Reset specific outbound tag for this slave
		query = query.Where("tag = ?", tag)
	}

	result := query.Updates(map[string]any{"up": 0, "down": 0, "total": 0})

	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
