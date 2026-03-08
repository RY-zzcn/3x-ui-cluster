package controller

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"strings"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/logger"
	"github.com/mhsanaei/3x-ui/v2/util/common"
	"github.com/mhsanaei/3x-ui/v2/web/service"
	"github.com/mhsanaei/3x-ui/v2/web/session"
	"github.com/mhsanaei/3x-ui/v2/web/websocket"

	"github.com/gin-gonic/gin"
)

// InboundController handles HTTP requests related to Xray inbounds management.
type InboundController struct {
	inboundService service.InboundService
	xrayService    service.XrayService
	slaveService   service.SlaveService
}

// NewInboundController creates a new InboundController and sets up its routes.
func NewInboundController(g *gin.RouterGroup) *InboundController {
	a := &InboundController{}
	a.initRouter(g)
	return a
}

// initRouter initializes the routes for inbound-related operations.
func (a *InboundController) initRouter(g *gin.RouterGroup) {

	g.GET("/list", a.getInbounds)
	g.GET("/get/:id", a.getInbound)
	g.GET("/:id/clients", a.getInboundClientEmails)
	g.GET("/getClientTraffics/:email", a.getClientTraffics)
	g.GET("/getClientTrafficsById/:id", a.getClientTrafficsById)

	g.POST("/add", a.addInbound)
	g.POST("/del/:id", a.delInbound)
	g.POST("/update/:id", a.updateInbound)
	g.POST("/clientIps/:email", a.getClientIps)
	g.POST("/clearClientIps/:email", a.clearClientIps)
	g.POST("/addClient", a.addInboundClient)
	g.POST("/:id/delClient/:clientId", a.delInboundClient)
	g.POST("/updateClient/:clientId", a.updateInboundClient)
	g.POST("/resetAllTraffics", a.resetAllTraffics)
	g.POST("/delDepletedClients/:id", a.delDepletedClients)
	g.POST("/import", a.importInbound)
	g.POST("/onlines", a.onlines)
	g.POST("/lastOnline", a.lastOnline)
	g.POST("/updateClientTraffic/:email", a.updateClientTraffic)
	g.POST("/:id/delClientByEmail/:email", a.delInboundClientByEmail)
}

// getInbounds retrieves the list of inbounds for the logged-in user.
// @Summary List inbounds
// @Description Returns all inbound configurations, optionally filtered by slave
// @Tags Inbounds
// @Produce json
// @Param slaveId query int false "Filter by slave ID (-1 for all)"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/list [get]
func (a *InboundController) getInbounds(c *gin.Context) {
	user := session.GetLoginUser(c)
	slaveIdStr := c.DefaultQuery("slaveId", "-1")
	slaveId, _ := strconv.Atoi(slaveIdStr)
	
	var inbounds []*model.Inbound
	var err error
	
	if slaveId == -1 {
		inbounds, err = a.inboundService.GetInbounds(user.Id)
	} else {
		inbounds, err = a.inboundService.GetInboundsForSlave(slaveId)
	}
	
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.obtain"), err)
		return
	}
	jsonObj(c, inbounds, nil)
}

// getInbound retrieves a specific inbound by its ID.
// @Summary Get inbound
// @Description Returns a specific inbound configuration by ID
// @Tags Inbounds
// @Produce json
// @Param id path int true "Inbound ID"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/get/{id} [get]
func (a *InboundController) getInbound(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	inbound, err := a.inboundService.GetInbound(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.obtain"), err)
		return
	}
	jsonObj(c, inbound, nil)
}

// getInboundClientEmails retrieves all client emails for a given inbound
// @Summary Get inbound client emails
// @Description Returns all client emails associated with an inbound
// @Tags Inbounds
// @Produce json
// @Param id path int true "Inbound ID"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/{id}/clients [get]
func (a *InboundController) getInboundClientEmails(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	
	emails, err := a.inboundService.GetInboundClients(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.obtain"), err)
		return
	}
	
	jsonObj(c, emails, nil)
}

// getClientTraffics retrieves client traffic information by email.
// @Summary Get client traffic by email
// @Description Returns traffic statistics for a client identified by email
// @Tags Inbounds
// @Produce json
// @Param email path string true "Client email"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/getClientTraffics/{email} [get]
func (a *InboundController) getClientTraffics(c *gin.Context) {
	email := c.Param("email")
	clientTraffics, err := a.inboundService.GetClientTrafficByEmail(email)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.trafficGetError"), err)
		return
	}
	jsonObj(c, clientTraffics, nil)
}

// getClientTrafficsById retrieves client traffic information by inbound ID.
// @Summary Get client traffic by ID
// @Description Returns traffic statistics for clients in an inbound
// @Tags Inbounds
// @Produce json
// @Param id path string true "Inbound ID"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/getClientTrafficsById/{id} [get]
func (a *InboundController) getClientTrafficsById(c *gin.Context) {
	id := c.Param("id")
	clientTraffics, err := a.inboundService.GetClientTrafficByID(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.trafficGetError"), err)
		return
	}
	jsonObj(c, clientTraffics, nil)
}

// addInbound creates a new inbound configuration.
// @Summary Add inbound
// @Description Creates a new inbound configuration on a slave
// @Tags Inbounds
// @Accept json
// @Produce json
// @Param inbound body model.Inbound true "Inbound configuration"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/add [post]
func (a *InboundController) addInbound(c *gin.Context) {
	inbound := &model.Inbound{}
	err := c.ShouldBind(inbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundCreateSuccess"), err)
		return
	}
	
	// Validate slave selection
	if inbound.SlaveId <= 0 {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), common.NewError("Please select a valid slave server"))
		return
	}
	
	user := session.GetLoginUser(c)
	inbound.UserId = user.Id

	// Generate tag with format inbound-<SlaveName>-<Protocol>-<Port>
	slaveName := "master"
	if inbound.SlaveId > 0 {
		slave, err := a.slaveService.GetSlave(inbound.SlaveId)
		if err == nil && slave != nil {
			slaveName = slave.Name
		} else {
			logger.Warningf("Failed to get slave name for id %d, using 'unknown'", inbound.SlaveId)
			slaveName = "unknown"
		}
	}
	// Sanitize slave name (replace spaces with dashes)
	slaveName = strings.ReplaceAll(slaveName, " ", "-")
	
	inbound.Tag = fmt.Sprintf("inbound-%s-%s-%d", slaveName, inbound.Protocol, inbound.Port)

	inbound, needRestart, err := a.inboundService.AddInbound(inbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.inbounds.toasts.inboundCreateSuccess"), inbound, nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
	if inbound.SlaveId > 0 {
		a.slaveService.PushConfig(inbound.SlaveId)
	}
	// Broadcast inbounds update via WebSocket
	inbounds, _ := a.inboundService.GetInbounds(user.Id)
	websocket.BroadcastInbounds(inbounds)
}

// delInbound deletes an inbound configuration by its ID.
// @Summary Delete inbound
// @Description Deletes an inbound configuration and pushes config to slave
// @Tags Inbounds
// @Produce json
// @Param id path int true "Inbound ID"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/del/{id} [post]
func (a *InboundController) delInbound(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundDeleteSuccess"), err)
		return
	}
    
    // Get inbound info before deletion to handle slave notification
    inbound, _ := a.inboundService.GetInbound(id)
    
	needRestart, err := a.inboundService.DelInbound(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.inbounds.toasts.inboundDeleteSuccess"), id, nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
    
    // Push config to slave if deleted inbound belonged to one
    if inbound != nil && inbound.SlaveId > 0 {
        a.slaveService.PushConfig(inbound.SlaveId)
    }

	// Broadcast inbounds update via WebSocket
	user := session.GetLoginUser(c)
	inbounds, _ := a.inboundService.GetInbounds(user.Id)
	websocket.BroadcastInbounds(inbounds)
}

// updateInbound updates an existing inbound configuration.
// @Summary Update inbound
// @Description Updates an existing inbound configuration
// @Tags Inbounds
// @Accept json
// @Produce json
// @Param id path int true "Inbound ID"
// @Param inbound body model.Inbound true "Updated inbound configuration"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/update/{id} [post]
func (a *InboundController) updateInbound(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}
	inbound, err := a.inboundService.GetInbound(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}

	// Backup original SlaveId for config push comparison
	originalSlaveId := inbound.SlaveId
	logger.Infof("Original SlaveId: %d", originalSlaveId)

	err = c.ShouldBindJSON(inbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}

	logger.Infof("New SlaveId after bind: %d", inbound.SlaveId)

	// Validate slave selection
	if inbound.SlaveId <= 0 {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), common.NewError("Please select a valid slave server"))
		return
	}

	inbound, needRestart, err := a.inboundService.UpdateInbound(inbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), inbound, nil)
	
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}

	if inbound.SlaveId > 0 {
		a.slaveService.PushConfig(inbound.SlaveId)
	}

    // If slave changed, push config to the original slave as well to remove the inbound
    if originalSlaveId > 0 && originalSlaveId != inbound.SlaveId {
        a.slaveService.PushConfig(originalSlaveId)
    }

	// Broadcast inbounds update via WebSocket
	user := session.GetLoginUser(c)
	inbounds, _ := a.inboundService.GetInbounds(user.Id)
	websocket.BroadcastInbounds(inbounds)
}

// getClientIps retrieves the IP addresses associated with a client by email.
// @Summary Get client IPs
// @Description Returns IP addresses associated with a client
// @Tags Inbounds
// @Produce json
// @Param email path string true "Client email"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/clientIps/{email} [post]
func (a *InboundController) getClientIps(c *gin.Context) {
	email := c.Param("email")

	ips, err := a.inboundService.GetInboundClientIps(email)
	if err != nil || ips == "" {
		jsonObj(c, "No IP Record", nil)
		return
	}

	// Prefer returning a normalized string list for consistent UI rendering
	type ipWithTimestamp struct {
		IP        string `json:"ip"`
		Timestamp int64  `json:"timestamp"`
	}

	var ipsWithTime []ipWithTimestamp
	if err := json.Unmarshal([]byte(ips), &ipsWithTime); err == nil && len(ipsWithTime) > 0 {
		formatted := make([]string, 0, len(ipsWithTime))
		for _, item := range ipsWithTime {
			if item.IP == "" {
				continue
			}
			if item.Timestamp > 0 {
				ts := time.Unix(item.Timestamp, 0).Local().Format("2006-01-02 15:04:05")
				formatted = append(formatted, fmt.Sprintf("%s (%s)", item.IP, ts))
				continue
			}
			formatted = append(formatted, item.IP)
		}
		jsonObj(c, formatted, nil)
		return
	}

	var oldIps []string
	if err := json.Unmarshal([]byte(ips), &oldIps); err == nil && len(oldIps) > 0 {
		jsonObj(c, oldIps, nil)
		return
	}

	// If parsing fails, return as string
	jsonObj(c, ips, nil)
}

// clearClientIps clears the IP addresses for a client by email.
// @Summary Clear client IPs
// @Description Clears all recorded IP addresses for a client
// @Tags Inbounds
// @Produce json
// @Param email path string true "Client email"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/clearClientIps/{email} [post]
func (a *InboundController) clearClientIps(c *gin.Context) {
	email := c.Param("email")

	err := a.inboundService.ClearClientIps(email)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.updateSuccess"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.logCleanSuccess"), nil)
}

// addInboundClient adds a new client to an existing inbound.
// @Summary Add client to inbound
// @Description Adds a new client to an existing inbound configuration
// @Tags Inbounds
// @Accept json
// @Produce json
// @Param client body model.Inbound true "Inbound with client data"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/addClient [post]
func (a *InboundController) addInboundClient(c *gin.Context) {
	data := &model.Inbound{}
	err := c.ShouldBind(data)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}

	needRestart, err := a.inboundService.AddInboundClient(data)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundClientAddSuccess"), nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
	// Push config to slave
	inbound, _ := a.inboundService.GetInbound(data.Id)
	if inbound != nil && inbound.SlaveId > 0 {
		a.slaveService.PushConfig(inbound.SlaveId)
	}
}

// delInboundClient deletes a client from an inbound by inbound ID and client ID.
// @Summary Delete inbound client
// @Description Removes a client from an inbound
// @Tags Inbounds
// @Produce json
// @Param id path int true "Inbound ID"
// @Param clientId path string true "Client ID"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/{id}/delClient/{clientId} [post]
func (a *InboundController) delInboundClient(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}
	clientId := c.Param("clientId")

	// Get inbound info before deletion
	inbound, _ := a.inboundService.GetInbound(id)

	needRestart, err := a.inboundService.DelInboundClient(id, clientId)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundClientDeleteSuccess"), nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
	// Push config to slave
	if inbound != nil && inbound.SlaveId > 0 {
		a.slaveService.PushConfig(inbound.SlaveId)
	}
}

// updateInboundClient updates a client's configuration in an inbound.
// @Summary Update inbound client
// @Description Updates a client's settings in an inbound
// @Tags Inbounds
// @Accept json
// @Produce json
// @Param clientId path string true "Client ID"
// @Param client body model.Inbound true "Inbound with updated client data"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/updateClient/{clientId} [post]
func (a *InboundController) updateInboundClient(c *gin.Context) {
	clientId := c.Param("clientId")

	inbound := &model.Inbound{}
	err := c.ShouldBind(inbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}

	needRestart, err := a.inboundService.UpdateInboundClient(inbound, clientId)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundClientUpdateSuccess"), nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
	// Push config to slave
	if inbound.SlaveId > 0 {
		a.slaveService.PushConfig(inbound.SlaveId)
	}
}

// resetAllTraffics resets all traffic counters across all inbounds.
// @Summary Reset all traffic
// @Description Resets traffic counters for all inbounds
// @Tags Inbounds
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/resetAllTraffics [post]
func (a *InboundController) resetAllTraffics(c *gin.Context) {
	err := a.inboundService.ResetAllTraffics()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	} else {
		a.xrayService.SetToNeedRestart()
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.resetAllTrafficSuccess"), nil)
}

// importInbound imports an inbound configuration from provided data.
// @Summary Import inbound
// @Description Imports an inbound configuration from JSON data
// @Tags Inbounds
// @Accept x-www-form-urlencoded
// @Produce json
// @Param data formData string true "Inbound JSON data"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/import [post]
func (a *InboundController) importInbound(c *gin.Context) {
	inbound := &model.Inbound{}
	err := json.Unmarshal([]byte(c.PostForm("data")), inbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	user := session.GetLoginUser(c)
	inbound.Id = 0
	inbound.UserId = user.Id
	
	// Generate tag with format inbound-<SlaveName>-<Protocol>-<Port>
	slaveName := "master"
	if inbound.SlaveId > 0 {
		slave, err := a.slaveService.GetSlave(inbound.SlaveId)
		if err == nil && slave != nil {
			slaveName = slave.Name
		} else {
			logger.Warningf("Failed to get slave name for id %d, using 'unknown'", inbound.SlaveId)
			slaveName = "unknown"
		}
	}
	// Sanitize slave name (replace spaces with dashes)
	slaveName = strings.ReplaceAll(slaveName, " ", "-")
	
	inbound.Tag = fmt.Sprintf("inbound-%s-%s-%d", slaveName, inbound.Protocol, inbound.Port)

	for index := range inbound.ClientStats {
		inbound.ClientStats[index].Id = 0
		inbound.ClientStats[index].Enable = true
	}

	needRestart := false
	inbound, needRestart, err = a.inboundService.AddInbound(inbound)
	jsonMsgObj(c, I18nWeb(c, "pages.inbounds.toasts.inboundCreateSuccess"), inbound, err)
	if err == nil && needRestart {
		a.xrayService.SetToNeedRestart()
	}
}

// delDepletedClients deletes clients in an inbound who have exhausted their traffic limits.
// @Summary Delete depleted clients
// @Description Removes clients whose traffic limits are exhausted
// @Tags Inbounds
// @Produce json
// @Param id path int true "Inbound ID"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/delDepletedClients/{id} [post]
func (a *InboundController) delDepletedClients(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}
	err = a.inboundService.DelDepletedClients(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.delDepletedClientsSuccess"), nil)
}

// onlines retrieves the list of currently online clients.
// @Summary Get online clients
// @Description Returns a list of currently online client emails
// @Tags Inbounds
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/onlines [post]
func (a *InboundController) onlines(c *gin.Context) {
	jsonObj(c, a.inboundService.GetOnlineClients(), nil)
}

// lastOnline retrieves the last online timestamps for clients.
// @Summary Get last online times
// @Description Returns last online timestamps for all clients
// @Tags Inbounds
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/lastOnline [post]
func (a *InboundController) lastOnline(c *gin.Context) {
	data, err := a.inboundService.GetClientsLastOnline()
	jsonObj(c, data, err)
}

// updateClientTraffic updates the traffic statistics for a client by email.
// @Summary Update client traffic
// @Description Sets upload/download traffic for a client
// @Tags Inbounds
// @Accept json
// @Produce json
// @Param email path string true "Client email"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/updateClientTraffic/{email} [post]
func (a *InboundController) updateClientTraffic(c *gin.Context) {
	email := c.Param("email")

	// Define the request structure for traffic update
	type TrafficUpdateRequest struct {
		Upload   int64 `json:"upload"`
		Download int64 `json:"download"`
	}

	var request TrafficUpdateRequest
	err := c.ShouldBindJSON(&request)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}

	err = a.inboundService.UpdateClientTrafficByEmail(email, request.Upload, request.Download)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}

	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundClientUpdateSuccess"), nil)
}

// delInboundClientByEmail deletes a client from an inbound by email address.
// @Summary Delete client by email
// @Description Removes a client from an inbound using their email
// @Tags Inbounds
// @Produce json
// @Param id path int true "Inbound ID"
// @Param email path string true "Client email"
// @Success 200 {object} entity.Msg
// @Router /panel/api/inbounds/{id}/delClientByEmail/{email} [post]
func (a *InboundController) delInboundClientByEmail(c *gin.Context) {
	inboundId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "Invalid inbound ID", err)
		return
	}

	email := c.Param("email")
	needRestart, err := a.inboundService.DelInboundClientByEmail(inboundId, email)
	if err != nil {
		jsonMsg(c, "Failed to delete client by email", err)
		return
	}

	jsonMsg(c, "Client deleted successfully", nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
}

