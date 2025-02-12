package controller

import (
	"x-ui/web/service"
	"github.com/gin-gonic/gin"
)

type OutboundController struct {
	outboundService service.OutboundService
	xrayService    service.XrayService
}

func NewOutboundController(g *gin.RouterGroup) *OutboundController {
	a := &OutboundController{}
	a.initRouter(g)
	return a
}

func (a *OutboundController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/outbound")

	g.POST("/list", a.getOutbounds)
	g.POST("/add", a.addOutbound)
	g.POST("/del/:tag", a.delOutbound)
	g.POST("/update/:tag", a.updateOutbound)
	g.POST("/resetTraffic/:tag", a.resetTraffic)
	g.POST("/resetAllTraffics", a.resetAllTraffics)
}

func (a *OutboundController) getOutbounds(c *gin.Context) {
	traffics, err := a.outboundService.GetOutboundsTraffic()
	if err != nil {
		jsonMsg(c, "Failed to get outbound list", err)
		return
	}
	jsonObj(c, traffics, nil)
}

func (a *OutboundController) addOutbound(c *gin.Context) {
	config, err := a.xrayService.GetXrayConfig()
	if err != nil {
		jsonMsg(c, "Failed to get configuration", err)
		return
	}

	var outbound map[string]interface{}
	err = c.ShouldBindJSON(&outbound)
	if err != nil {
		jsonMsg(c, "Failed to parse outbound configuration", err)
		return
	}

	// Validate required fields
	if tag, ok := outbound["tag"].(string); !ok || tag == "" {
		jsonMsg(c, "Invalid or missing tag in outbound configuration", nil)
		return
	}
	if _, ok := outbound["protocol"].(string); !ok {
		jsonMsg(c, "Invalid or missing protocol in outbound configuration", nil)
		return
	}

	// Add the outbound configuration
	config.OutboundConfigs = append(config.OutboundConfigs, json_util.ToRawMessage(outbound))

	err = a.xrayService.SetXrayConfig(config)
	if err != nil {
		jsonMsg(c, "Failed to update configuration", err)
		return
	}

	jsonMsg(c, "Outbound added successfully", nil)
}

func (a *OutboundController) delOutbound(c *gin.Context) {
	config, err := a.xrayService.GetXrayConfig()
	if err != nil {
		jsonMsg(c, "Failed to get configuration", err)
		return
	}

	tag := c.Param("tag")
	if tag == "" {
		jsonMsg(c, "Invalid outbound tag", nil)
		return
	}

	var found bool
	var newOutbounds []json_util.RawMessage
	for _, outbound := range config.OutboundConfigs {
		var ob map[string]interface{}
		if err := json.Unmarshal(outbound, &ob); err != nil {
			continue
		}
		if ob["tag"] == tag {
			found = true
			continue
		}
		newOutbounds = append(newOutbounds, outbound)
	}

	if !found {
		jsonMsg(c, "Outbound not found", nil)
		return
	}

	config.OutboundConfigs = newOutbounds
	err = a.xrayService.SetXrayConfig(config)
	if err != nil {
		jsonMsg(c, "Failed to update configuration", err)
		return
	}

	jsonMsg(c, "Outbound deleted successfully", nil)
}

func (a *OutboundController) updateOutbound(c *gin.Context) {
	config, err := a.xrayService.GetXrayConfig()
	if err != nil {
		jsonMsg(c, "Failed to get configuration", err)
		return
	}

	tag := c.Param("tag")
	if tag == "" {
		jsonMsg(c, "Invalid outbound tag", nil)
		return
	}

	var newOutbound map[string]interface{}
	err = c.ShouldBindJSON(&newOutbound)
	if err != nil {
		jsonMsg(c, "Failed to parse outbound configuration", err)
		return
	}

	// Validate required fields
	if newTag, ok := newOutbound["tag"].(string); !ok || newTag == "" {
		jsonMsg(c, "Invalid or missing tag in outbound configuration", nil)
		return
	}
	if _, ok := newOutbound["protocol"].(string); !ok {
		jsonMsg(c, "Invalid or missing protocol in outbound configuration", nil)
		return
	}

	var found bool
	var newOutbounds []json_util.RawMessage
	for _, outbound := range config.OutboundConfigs {
		var ob map[string]interface{}
		if err := json.Unmarshal(outbound, &ob); err != nil {
			continue
		}
		if ob["tag"] == tag {
			found = true
			newOutbounds = append(newOutbounds, json_util.ToRawMessage(newOutbound))
		} else {
			newOutbounds = append(newOutbounds, outbound)
		}
	}

	if !found {
		jsonMsg(c, "Outbound not found", nil)
		return
	}

	config.OutboundConfigs = newOutbounds
	err = a.xrayService.SetXrayConfig(config)
	if err != nil {
		jsonMsg(c, "Failed to update configuration", err)
		return
	}

	jsonMsg(c, "Outbound updated successfully", nil)
}

func (a *OutboundController) resetTraffic(c *gin.Context) {
	tag := c.Param("tag")
	if tag == "" {
		jsonMsg(c, "Invalid outbound tag", nil)
		return
	}

	err := a.outboundService.ResetTraffic(tag)
	if err != nil {
		jsonMsg(c, "Failed to reset traffic", err)
		return
	}
	jsonMsg(c, "Traffic reset successfully", nil)
}

func (a *OutboundController) resetAllTraffics(c *gin.Context) {
	err := a.outboundService.ResetAllTraffics()
	if err != nil {
		jsonMsg(c, "Failed to reset all traffic", err)
		return
	}
	jsonMsg(c, "All traffic reset successfully", nil)
}
