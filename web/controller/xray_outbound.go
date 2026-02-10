package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v2/web/service"
)

type OutboundController struct {
	outboundService service.OutboundService
}

func NewOutboundController(g *gin.RouterGroup) *OutboundController {
	a := &OutboundController{}
	a.initRouter(g)
	return a
}

func (a *OutboundController) initRouter(g *gin.RouterGroup) {
	g.GET("/list", a.getOutbounds)
	g.POST("/add", a.addOutbound)
	g.POST("/update", a.updateOutbound)
	g.POST("/del/:id", a.deleteOutbound)
}

func (a *OutboundController) getSlaveId(c *gin.Context) (int, error) {
	slaveIdStr := c.Query("slaveId")
	if slaveIdStr == "" {
		slaveIdStr = c.PostForm("slaveId")
	}
	if slaveIdStr == "" {
		return 0, nil
	}
	return strconv.Atoi(slaveIdStr)
}

func (a *OutboundController) getOutbounds(c *gin.Context) {
	slaveId, err := a.getSlaveId(c)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "error"), err)
		return
	}
	if slaveId <= 0 {
		jsonMsg(c, I18nWeb(c, "error"), nil)
		return
	}

	list, err := a.outboundService.GetOutbounds(slaveId)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.getSettings"), err)
		return
	}
	jsonObj(c, list, nil)
}

func (a *OutboundController) addOutbound(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, I18nWeb(c, "error"), err)
		return
	}

	// Extract slaveId from request body
	slaveIdFloat, ok := req["slaveId"].(float64)
	if !ok || int(slaveIdFloat) <= 0 {
		jsonMsg(c, "slaveId is required", nil)
		return
	}
	slaveId := int(slaveIdFloat)
	delete(req, "slaveId")

	err := a.outboundService.AddOutbound(slaveId, req)
	if err == nil {
		go a.pushConfigToSlave(slaveId)
	}
	jsonMsg(c, I18nWeb(c, "success"), err)
}

func (a *OutboundController) updateOutbound(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, I18nWeb(c, "error"), err)
		return
	}

	// Extract slaveId
	slaveIdFloat, ok := req["slaveId"].(float64)
	if !ok || int(slaveIdFloat) <= 0 {
		jsonMsg(c, "slaveId is required", nil)
		return
	}
	slaveId := int(slaveIdFloat)
	delete(req, "slaveId")

	// Extract index from the "id" field
	idFloat, ok := req["id"].(float64)
	if !ok {
		jsonMsg(c, I18nWeb(c, "error"), nil)
		return
	}
	index := int(idFloat)

	err := a.outboundService.UpdateOutbound(slaveId, index, req)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.modifySettings"), err)
		return
	}

	go a.pushConfigToSlave(slaveId)
	jsonMsg(c, I18nWeb(c, "success"), nil)
}

func (a *OutboundController) deleteOutbound(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "error"), err)
		return
	}

	slaveId, err := a.getSlaveId(c)
	if err != nil || slaveId <= 0 {
		jsonMsg(c, "slaveId is required", err)
		return
	}

	err = a.outboundService.DeleteOutbound(slaveId, id)
	if err == nil {
		go a.pushConfigToSlave(slaveId)
	}
	jsonMsg(c, I18nWeb(c, "success"), err)
}

// pushConfigToSlave pushes the updated config to a specific slave
func (a *OutboundController) pushConfigToSlave(slaveId int) {
	slaveService := service.SlaveService{}
	slaveService.PushConfig(slaveId)
}
