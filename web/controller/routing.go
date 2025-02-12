package controller

import (
	"encoding/json"
	"fmt"
	"x-ui/web/service"
	"github.com/gin-gonic/gin"
)

type RoutingController struct {
	xrayService service.XrayService
}

func NewRoutingController(g *gin.RouterGroup) *RoutingController {
	a := &RoutingController{}
	a.initRouter(g)
	return a
}

func (a *RoutingController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/routing")

	g.POST("/get", a.getRouting)
	g.POST("/update", a.updateRouting)
	g.POST("/rule/add", a.addRule)
	g.POST("/rule/del", a.deleteRule)
	g.POST("/rule/update", a.updateRule)
}

func (a *RoutingController) getRouting(c *gin.Context) {
	config, err := a.xrayService.GetXrayConfig()
	if err != nil {
		jsonMsg(c, "Failed to get routing configuration", err)
		return
	}
	jsonObj(c, config.RouterConfig, nil)
}

func (a *RoutingController) validateRoutingConfig(routing map[string]interface{}) error {
	// Validate domainStrategy
	if strategy, ok := routing["domainStrategy"].(string); ok {
		validStrategies := map[string]bool{
			"AsIs":     true,
			"IPIfNonMatch": true,
			"IPOnDemand":   true,
		}
		if !validStrategies[strategy] {
			return fmt.Errorf("invalid domainStrategy: %s", strategy)
		}
	}

	// Validate rules
	if rules, ok := routing["rules"].([]interface{}); ok {
		for _, rule := range rules {
			ruleMap, ok := rule.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid rule format")
			}

			// Check required fields
			if _, ok := ruleMap["type"].(string); !ok {
				return fmt.Errorf("rule missing type field")
			}

			// Validate outboundTag
			if tag, ok := ruleMap["outboundTag"].(string); ok && tag == "" {
				return fmt.Errorf("rule has empty outboundTag")
			}
		}
	}

	return nil
}

func (a *RoutingController) updateRouting(c *gin.Context) {
	config, err := a.xrayService.GetXrayConfig()
	if err != nil {
		jsonMsg(c, "Failed to get configuration", err)
		return
	}

	var newRouting map[string]interface{}
	err = c.ShouldBindJSON(&newRouting)
	if err != nil {
		jsonMsg(c, "Failed to parse routing configuration", err)
		return
	}

	// Validate routing configuration
	err = a.validateRoutingConfig(newRouting)
	if err != nil {
		jsonMsg(c, "Invalid routing configuration: "+err.Error(), nil)
		return
	}

	config.RouterConfig = json_util.ToRawMessage(newRouting)
	err = a.xrayService.SetXrayConfig(config)
	if err != nil {
		jsonMsg(c, "Failed to update configuration", err)
		return
	}

	jsonMsg(c, "Routing configuration updated successfully", nil)
}

func (a *RoutingController) addRule(c *gin.Context) {
	config, err := a.xrayService.GetXrayConfig()
	if err != nil {
		jsonMsg(c, "Failed to get configuration", err)
		return
	}

	var newRule map[string]interface{}
	err = c.ShouldBindJSON(&newRule)
	if err != nil {
		jsonMsg(c, "Failed to parse rule configuration", err)
		return
	}

	// Validate the new rule
	routing := make(map[string]interface{})
	json.Unmarshal(config.RouterConfig, &routing)
	routing["rules"] = append(routing["rules"].([]interface{}), newRule)

	err = a.validateRoutingConfig(routing)
	if err != nil {
		jsonMsg(c, "Invalid rule configuration: "+err.Error(), nil)
		return
	}

	config.RouterConfig = json_util.ToRawMessage(routing)
	err = a.xrayService.SetXrayConfig(config)
	if err != nil {
		jsonMsg(c, "Failed to update configuration", err)
		return
	}

	jsonMsg(c, "Rule added successfully", nil)
}

func (a *RoutingController) deleteRule(c *gin.Context) {
	config, err := a.xrayService.GetXrayConfig()
	if err != nil {
		jsonMsg(c, "Failed to get configuration", err)
		return
	}

	var index int
	err = c.ShouldBindJSON(&index)
	if err != nil {
		jsonMsg(c, "Failed to parse rule index", err)
		return
	}

	routing := make(map[string]interface{})
	json.Unmarshal(config.RouterConfig, &routing)

	rules := routing["rules"].([]interface{})
	if index < 0 || index >= len(rules) {
		jsonMsg(c, "Invalid rule index", nil)
		return
	}

	// Remove the rule at the specified index
	newRules := append(rules[:index], rules[index+1:]...)
	routing["rules"] = newRules

	config.RouterConfig = json_util.ToRawMessage(routing)
	err = a.xrayService.SetXrayConfig(config)
	if err != nil {
		jsonMsg(c, "Failed to update configuration", err)
		return
	}

	jsonMsg(c, "Rule deleted successfully", nil)
}

func (a *RoutingController) updateRule(c *gin.Context) {
	config, err := a.xrayService.GetXrayConfig()
	if err != nil {
		jsonMsg(c, "Failed to get configuration", err)
		return
	}

	type UpdateRuleRequest struct {
		Index int                    `json:"index"`
		Rule  map[string]interface{} `json:"rule"`
	}

	var req UpdateRuleRequest
	err = c.ShouldBindJSON(&req)
	if err != nil {
		jsonMsg(c, "Failed to parse request", err)
		return
	}

	routing := make(map[string]interface{})
	json.Unmarshal(config.RouterConfig, &routing)

	rules := routing["rules"].([]interface{})
	if req.Index < 0 || req.Index >= len(rules) {
		jsonMsg(c, "Invalid rule index", nil)
		return
	}

	// Update the rule at the specified index
	rules[req.Index] = req.Rule
	routing["rules"] = rules

	// Validate the updated configuration
	err = a.validateRoutingConfig(routing)
	if err != nil {
		jsonMsg(c, "Invalid rule configuration: "+err.Error(), nil)
		return
	}

	config.RouterConfig = json_util.ToRawMessage(routing)
	err = a.xrayService.SetXrayConfig(config)
	if err != nil {
		jsonMsg(c, "Failed to update configuration", err)
		return
	}

	jsonMsg(c, "Rule updated successfully", nil)
}
