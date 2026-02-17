package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v2/logger"
	"github.com/mhsanaei/3x-ui/v2/web/service"
)

type RoutingController struct {
	routingService service.RoutingService
}

func NewRoutingController(g *gin.RouterGroup) *RoutingController {
	a := &RoutingController{}
	a.initRouter(g)
	return a
}

func (a *RoutingController) initRouter(g *gin.RouterGroup) {
	g.GET("/list", a.getRoutingRules)
	g.POST("/add", a.addRoutingRule)
	g.POST("/update", a.updateRoutingRule)
	g.POST("/del/:id", a.deleteRoutingRule)
}

func (a *RoutingController) getSlaveId(c *gin.Context) (int, error) {
	slaveIdStr := c.Query("slaveId")
	if slaveIdStr == "" {
		slaveIdStr = c.PostForm("slaveId")
	}
	if slaveIdStr == "" {
		return 0, nil
	}
	return strconv.Atoi(slaveIdStr)
}

func (a *RoutingController) getRoutingRules(c *gin.Context) {
	slaveId, err := a.getSlaveId(c)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "error"), err)
		return
	}
	if slaveId <= 0 {
		jsonMsg(c, I18nWeb(c, "error"), nil)
		return
	}

	list, err := a.routingService.GetRoutingRules(slaveId)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.getSettings"), err)
		return
	}
	jsonObj(c, list, nil)
}

func (a *RoutingController) addRoutingRule(c *gin.Context) {
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

	err := a.routingService.AddRoutingRule(slaveId, req)
	if err == nil {
		go a.pushConfigToSlave(slaveId)
	}
	jsonMsg(c, I18nWeb(c, "success"), err)
}

func (a *RoutingController) updateRoutingRule(c *gin.Context) {
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

	err := a.routingService.UpdateRoutingRule(slaveId, index, req)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.modifySettings"), err)
		return
	}

	go a.pushConfigToSlave(slaveId)
	jsonMsg(c, I18nWeb(c, "success"), nil)
}

func (a *RoutingController) deleteRoutingRule(c *gin.Context) {
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

	err = a.routingService.DeleteRoutingRule(slaveId, id)
	if err == nil {
		go a.pushConfigToSlave(slaveId)
	}
	jsonMsg(c, I18nWeb(c, "success"), err)
}

// pushConfigToSlave pushes the updated config to a specific slave
// pushConfigToSlave pushes the updated config to a specific slave
func (a *RoutingController) pushConfigToSlave(slaveId int) {
	logger.Infof("RoutingController: pushing config to slave %d", slaveId)
	slaveService := service.SlaveService{}
	err := slaveService.PushConfig(slaveId)
	if err != nil {
		logger.Errorf("RoutingController: failed to push config to slave %d: %v", slaveId, err)
	} else {
		logger.Infof("RoutingController: successfully pushed config to slave %d", slaveId)
	}
}
