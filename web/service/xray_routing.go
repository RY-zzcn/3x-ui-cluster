package service

import (
	"encoding/json"
	"fmt"

	"github.com/mhsanaei/3x-ui/v2/logger"
)

// RoutingService provides business logic for managing Xray routing rules.
// Routing rules are stored directly in the xrayTemplateConfig JSON in the slave_settings table.
type RoutingService struct {
	SlaveSettingService SlaveSettingService
}

// getTemplateRoutingRules parses the xrayTemplateConfig for a slave and returns the routing.rules array
func (s *RoutingService) getTemplateRoutingRules(slaveId int) ([]map[string]interface{}, error) {
	templateJson, err := s.SlaveSettingService.GetXrayConfigForSlave(slaveId)
	if err != nil {
		return nil, fmt.Errorf("failed to get xray template config for slave %d: %v", slaveId, err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(templateJson), &config); err != nil {
		return nil, fmt.Errorf("failed to parse xray template config: %v", err)
	}

	routingRaw, ok := config["routing"]
	if !ok {
		return []map[string]interface{}{}, nil
	}

	routing, ok := routingRaw.(map[string]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	rulesRaw, ok := routing["rules"]
	if !ok {
		return []map[string]interface{}{}, nil
	}

	rulesArr, ok := rulesRaw.([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	result := make([]map[string]interface{}, 0, len(rulesArr))
	for _, item := range rulesArr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result, nil
}

// saveTemplateRoutingRules updates the routing.rules array in xrayTemplateConfig for a slave and saves it
func (s *RoutingService) saveTemplateRoutingRules(slaveId int, rules []map[string]interface{}) error {
	templateJson, err := s.SlaveSettingService.GetXrayConfigForSlave(slaveId)
	if err != nil {
		return fmt.Errorf("failed to get xray template config for slave %d: %v", slaveId, err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(templateJson), &config); err != nil {
		return fmt.Errorf("failed to parse xray template config: %v", err)
	}

	// Ensure routing section exists
	routingRaw, ok := config["routing"]
	if !ok {
		config["routing"] = map[string]interface{}{
			"domainStrategy": "AsIs",
			"rules":          rules,
		}
	} else {
		routing, ok := routingRaw.(map[string]interface{})
		if !ok {
			routing = map[string]interface{}{"domainStrategy": "AsIs"}
		}
		routing["rules"] = rules
		config["routing"] = routing
	}

	newJson, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal xray template config: %v", err)
	}

	return s.SlaveSettingService.SaveXrayConfigForSlave(slaveId, string(newJson))
}

// GetRoutingRules returns all routing rules from the template config for a slave.
// Each rule is returned with an "id" field set to its array index.
func (s *RoutingService) GetRoutingRules(slaveId int) ([]map[string]interface{}, error) {
	rules, err := s.getTemplateRoutingRules(slaveId)
	if err != nil {
		return nil, err
	}

	// Add pseudo-ID (array index) for frontend
	for i := range rules {
		rules[i]["id"] = i
	}
	return rules, nil
}

// AddRoutingRule adds a new routing rule to the template config for a slave
func (s *RoutingService) AddRoutingRule(slaveId int, rule map[string]interface{}) error {
	rules, err := s.getTemplateRoutingRules(slaveId)
	if err != nil {
		return err
	}

	// Remove any frontend-generated id
	delete(rule, "id")

	rules = append(rules, rule)
	logger.Infof("Added routing rule for slave %d, total rules: %d", slaveId, len(rules))
	return s.saveTemplateRoutingRules(slaveId, rules)
}

// UpdateRoutingRule updates a routing rule at the given index in the template config for a slave
func (s *RoutingService) UpdateRoutingRule(slaveId int, index int, rule map[string]interface{}) error {
	rules, err := s.getTemplateRoutingRules(slaveId)
	if err != nil {
		return err
	}

	if index < 0 || index >= len(rules) {
		return fmt.Errorf("routing rule index %d out of range (total: %d)", index, len(rules))
	}

	// Remove any frontend-generated id
	delete(rule, "id")

	rules[index] = rule
	logger.Infof("Updated routing rule at index %d for slave %d", index, slaveId)
	return s.saveTemplateRoutingRules(slaveId, rules)
}

// DeleteRoutingRule removes a routing rule at the given index from the template config for a slave
func (s *RoutingService) DeleteRoutingRule(slaveId int, index int) error {
	rules, err := s.getTemplateRoutingRules(slaveId)
	if err != nil {
		return err
	}

	if index < 0 || index >= len(rules) {
		return fmt.Errorf("routing rule index %d out of range (total: %d)", index, len(rules))
	}

	rules = append(rules[:index], rules[index+1:]...)
	logger.Infof("Deleted routing rule at index %d for slave %d, remaining: %d", index, slaveId, len(rules))
	return s.saveTemplateRoutingRules(slaveId, rules)
}
