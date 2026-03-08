package controller

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/mhsanaei/3x-ui/v2/web/global"
	"github.com/mhsanaei/3x-ui/v2/web/service"
	"github.com/mhsanaei/3x-ui/v2/web/websocket"

	"github.com/gin-gonic/gin"
)

var filenameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-.]+$`)

// ServerController handles server management and status-related operations.
type ServerController struct {
	BaseController

	serverService  service.ServerService
	settingService service.SettingService

	lastStatus *service.Status

	lastVersions        []string
	lastGetVersionsTime int64 // unix seconds
}

// NewServerController creates a new ServerController, initializes routes, and starts background tasks.
func NewServerController(g *gin.RouterGroup) *ServerController {
	a := &ServerController{}
	a.initRouter(g)
	a.startTask()
	return a
}

// initRouter sets up the routes for server status, Xray management, and utility endpoints.
func (a *ServerController) initRouter(g *gin.RouterGroup) {

	g.GET("/status", a.status)
	g.GET("/cpuHistory/:bucket", a.getCpuHistoryBucket)
	g.GET("/getXrayVersion", a.getXrayVersion)
	g.GET("/getConfigJson", a.getConfigJson)
	g.GET("/getDb", a.getDb)
	g.GET("/getNewUUID", a.getNewUUID)
	g.GET("/getNewX25519Cert", a.getNewX25519Cert)
	g.GET("/getNewmldsa65", a.getNewmldsa65)
	g.GET("/getNewmlkem768", a.getNewmlkem768)
	g.GET("/getNewVlessEnc", a.getNewVlessEnc)

	g.POST("/stopXrayService", a.stopXrayService)
	g.POST("/restartXrayService", a.restartXrayService)
	g.POST("/restartSlaveXray/:slaveId", a.restartSlaveXray)
	g.POST("/installXray/:version", a.installXray)
	g.POST("/updateGeofile", a.updateGeofile)
	g.POST("/updateGeofile/:fileName", a.updateGeofile)
	g.POST("/logs/:count", a.getLogs)
	g.POST("/xraylogs/:count", a.getXrayLogs)
	g.POST("/importDB", a.importDB)
	g.POST("/getNewEchCert", a.getNewEchCert)
}

// refreshStatus updates the cached server status and collects CPU history.
func (a *ServerController) refreshStatus() {
	a.lastStatus = a.serverService.GetStatus(a.lastStatus)
	// collect cpu history when status is fresh
	if a.lastStatus != nil {
		a.serverService.AppendCpuSample(time.Now(), a.lastStatus.Cpu)
		// Broadcast status update via WebSocket
		websocket.BroadcastStatus(a.lastStatus)
	}
}

// startTask initiates background tasks for continuous status monitoring.
func (a *ServerController) startTask() {
	webServer := global.GetWebServer()
	c := webServer.GetCron()
	c.AddFunc("@every 2s", func() {
		// Always refresh to keep CPU history collected continuously.
		// Sampling is lightweight and capped to ~6 hours in memory.
		a.refreshStatus()
	})
}

// status returns the current server status information.
// @Summary Get server status
// @Description Returns the current server status including CPU, memory, disk, and network info
// @Tags Server
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/status [get]
func (a *ServerController) status(c *gin.Context) { jsonObj(c, a.lastStatus, nil) }

// getCpuHistoryBucket retrieves aggregated CPU usage history based on the specified time bucket.
// @Summary Get CPU history
// @Description Returns aggregated CPU usage history for chart display
// @Tags Server
// @Produce json
// @Param bucket path int true "Time bucket in seconds (2,30,60,120,180,300)"
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/cpuHistory/{bucket} [get]
func (a *ServerController) getCpuHistoryBucket(c *gin.Context) {
	bucketStr := c.Param("bucket")
	bucket, err := strconv.Atoi(bucketStr)
	if err != nil || bucket <= 0 {
		jsonMsg(c, "invalid bucket", fmt.Errorf("bad bucket"))
		return
	}
	allowed := map[int]bool{
		2:   true, // Real-time view
		30:  true, // 30s intervals
		60:  true, // 1m intervals
		120: true, // 2m intervals
		180: true, // 3m intervals
		300: true, // 5m intervals
	}
	if !allowed[bucket] {
		jsonMsg(c, "invalid bucket", fmt.Errorf("unsupported bucket"))
		return
	}
	points := a.serverService.AggregateCpuHistory(bucket, 60)
	jsonObj(c, points, nil)
}

// getXrayVersion retrieves available Xray versions, with caching for 1 minute.
// @Summary Get Xray versions
// @Description Returns a list of available Xray versions (cached for 1 minute)
// @Tags Server
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/getXrayVersion [get]
func (a *ServerController) getXrayVersion(c *gin.Context) {
	now := time.Now().Unix()
	if now-a.lastGetVersionsTime <= 60 { // 1 minute cache
		jsonObj(c, a.lastVersions, nil)
		return
	}

	versions, err := a.serverService.GetXrayVersions()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "getVersion"), err)
		return
	}

	a.lastVersions = versions
	a.lastGetVersionsTime = now

	jsonObj(c, versions, nil)
}

// installXray installs or updates Xray to the specified version.
// @Summary Install Xray version
// @Description Installs or updates Xray to the specified version
// @Tags Server
// @Produce json
// @Param version path string true "Xray version to install"
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/installXray/{version} [post]
func (a *ServerController) installXray(c *gin.Context) {
	version := c.Param("version")
	err := a.serverService.UpdateXray(version)
	jsonMsg(c, I18nWeb(c, "pages.index.xraySwitchVersionPopover"), err)
}

// updateGeofile updates the specified geo file for Xray.
// @Summary Update geo file
// @Description Updates the specified geo file (geoip.dat/geosite.dat) for Xray
// @Tags Server
// @Produce json
// @Param fileName path string false "Geo file name"
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/updateGeofile/{fileName} [post]
func (a *ServerController) updateGeofile(c *gin.Context) {
	fileName := c.Param("fileName")

	// Validate the filename for security (prevent path traversal attacks)
	if fileName != "" && !a.serverService.IsValidGeofileName(fileName) {
		jsonMsg(c, I18nWeb(c, "pages.index.geofileUpdatePopover"),
			fmt.Errorf("invalid filename: contains unsafe characters or path traversal patterns"))
		return
	}

	err := a.serverService.UpdateGeofile(fileName)
	jsonMsg(c, I18nWeb(c, "pages.index.geofileUpdatePopover"), err)
}

// stopXrayService stops the Xray service.
// @Summary Stop Xray service
// @Description Stops the running Xray service
// @Tags Server
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/stopXrayService [post]
func (a *ServerController) stopXrayService(c *gin.Context) {
	err := a.serverService.StopXrayService()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.xray.stopError"), err)
		websocket.BroadcastXrayState("error", err.Error())
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.xray.stopSuccess"), err)
	websocket.BroadcastXrayState("stop", "")
	websocket.BroadcastNotification(
		I18nWeb(c, "pages.xray.stopSuccess"),
		"Xray service has been stopped",
		"warning",
	)
}

// restartXrayService restarts the Xray service.
// @Summary Restart Xray service
// @Description Restarts the Xray service
// @Tags Server
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/restartXrayService [post]
func (a *ServerController) restartXrayService(c *gin.Context) {
	err := a.serverService.RestartXrayService()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.xray.restartError"), err)
		websocket.BroadcastXrayState("error", err.Error())
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.xray.restartSuccess"), err)
	websocket.BroadcastXrayState("running", "")
	websocket.BroadcastNotification(
		I18nWeb(c, "pages.xray.restartSuccess"),
		"Xray service has been restarted successfully",
		"success",
	)
}

// restartSlaveXray restarts the Xray service on a specific slave.
// @Summary Restart slave Xray
// @Description Restarts the Xray service on a specific slave node
// @Tags Server
// @Produce json
// @Param slaveId path int true "Slave ID"
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/restartSlaveXray/{slaveId} [post]
func (a *ServerController) restartSlaveXray(c *gin.Context) {
	slaveId, err := strconv.Atoi(c.Param("slaveId"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.xray.restartError"), err)
		return
	}

	slaveService := &service.SlaveService{}
	err = slaveService.RestartSlaveXray(slaveId)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.xray.restartError"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.xray.restartSuccess"), err)
}

// getLogs retrieves the application logs based on count, level, and syslog filters.
// @Summary Get application logs
// @Description Retrieves panel logs with filtering by count, level, and syslog
// @Tags Server
// @Accept x-www-form-urlencoded
// @Produce json
// @Param count path string true "Number of log lines"
// @Param level formData string false "Log level filter"
// @Param syslog formData string false "Syslog filter"
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/logs/{count} [post]
func (a *ServerController) getLogs(c *gin.Context) {
	count := c.Param("count")
	level := c.PostForm("level")
	syslog := c.PostForm("syslog")
	logs := a.serverService.GetLogs(count, level, syslog)
	jsonObj(c, logs, nil)
}

// getXrayLogs retrieves Xray logs with filtering options for direct, blocked, and proxy traffic.
// @Summary Get Xray logs
// @Description Retrieves Xray access logs with filtering by traffic type
// @Tags Server
// @Accept x-www-form-urlencoded
// @Produce json
// @Param count path string true "Number of log lines"
// @Param filter formData string false "Text filter"
// @Param showDirect formData string false "Show direct traffic"
// @Param showBlocked formData string false "Show blocked traffic"
// @Param showProxy formData string false "Show proxy traffic"
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/xraylogs/{count} [post]
func (a *ServerController) getXrayLogs(c *gin.Context) {
	count := c.Param("count")
	filter := c.PostForm("filter")
	showDirect := c.PostForm("showDirect")
	showBlocked := c.PostForm("showBlocked")
	showProxy := c.PostForm("showProxy")

	var freedoms []string
	var blackholes []string

	//getting tags for freedom and blackhole outbounds
	config, err := a.settingService.GetDefaultXrayConfig()
	if err == nil && config != nil {
		if cfgMap, ok := config.(map[string]any); ok {
			if outbounds, ok := cfgMap["outbounds"].([]any); ok {
				for _, outbound := range outbounds {
					if obMap, ok := outbound.(map[string]any); ok {
						switch obMap["protocol"] {
						case "freedom":
							if tag, ok := obMap["tag"].(string); ok {
								freedoms = append(freedoms, tag)
							}
						case "blackhole":
							if tag, ok := obMap["tag"].(string); ok {
								blackholes = append(blackholes, tag)
							}
						}
					}
				}
			}
		}
	}

	if len(freedoms) == 0 {
		freedoms = []string{"direct"}
	}
	if len(blackholes) == 0 {
		blackholes = []string{"blocked"}
	}

	logs := a.serverService.GetXrayLogs(count, filter, showDirect, showBlocked, showProxy, freedoms, blackholes)
	jsonObj(c, logs, nil)
}

// getConfigJson retrieves the Xray configuration as JSON.
// @Summary Get Xray config JSON
// @Description Returns the current Xray configuration as JSON
// @Tags Server
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/getConfigJson [get]
func (a *ServerController) getConfigJson(c *gin.Context) {
	configJson, err := a.serverService.GetConfigJson()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.index.getConfigError"), err)
		return
	}
	jsonObj(c, configJson, nil)
}

// getDb downloads the database file.
// @Summary Download database
// @Description Downloads the panel database file
// @Tags Server
// @Produce octet-stream
// @Success 200 {file} file
// @Router /panel/api/server/getDb [get]
func (a *ServerController) getDb(c *gin.Context) {
	db, err := a.serverService.GetDb()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.index.getDatabaseError"), err)
		return
	}

	filename := "x-ui.db"

	if !isValidFilename(filename) {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("invalid filename"))
		return
	}

	// Set the headers for the response
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename="+filename)

	// Write the file contents to the response
	c.Writer.Write(db)
}

func isValidFilename(filename string) bool {
	// Validate that the filename only contains allowed characters
	return filenameRegex.MatchString(filename)
}

// importDB imports a database file and restarts the Xray service.
// @Summary Import database
// @Description Imports a database file and restarts the Xray service
// @Tags Server
// @Accept multipart/form-data
// @Produce json
// @Param db formData file true "Database file"
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/importDB [post]
func (a *ServerController) importDB(c *gin.Context) {
	// Get the file from the request body
	file, _, err := c.Request.FormFile("db")
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.index.readDatabaseError"), err)
		return
	}
	defer file.Close()
	// Always restart Xray before return
	defer a.serverService.RestartXrayService()
	// lastGetStatusTime removed; no longer needed
	// Import it
	err = a.serverService.ImportDB(file)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.index.importDatabaseError"), err)
		return
	}
	jsonObj(c, I18nWeb(c, "pages.index.importDatabaseSuccess"), nil)
}

// getNewX25519Cert generates a new X25519 certificate.
// @Summary Generate X25519 certificate
// @Description Generates a new X25519 key pair
// @Tags Server
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/getNewX25519Cert [get]
func (a *ServerController) getNewX25519Cert(c *gin.Context) {
	cert, err := a.serverService.GetNewX25519Cert()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.getNewX25519CertError"), err)
		return
	}
	jsonObj(c, cert, nil)
}

// getNewmldsa65 generates a new ML-DSA-65 key.
// @Summary Generate ML-DSA-65 key
// @Description Generates a new ML-DSA-65 key pair
// @Tags Server
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/getNewmldsa65 [get]
func (a *ServerController) getNewmldsa65(c *gin.Context) {
	cert, err := a.serverService.GetNewmldsa65()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.getNewmldsa65Error"), err)
		return
	}
	jsonObj(c, cert, nil)
}

// getNewEchCert generates a new ECH certificate for the given SNI.
// @Summary Generate ECH certificate
// @Description Generates a new ECH (Encrypted Client Hello) certificate
// @Tags Server
// @Accept x-www-form-urlencoded
// @Produce json
// @Param sni formData string true "Server Name Indication"
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/getNewEchCert [post]
func (a *ServerController) getNewEchCert(c *gin.Context) {
	sni := c.PostForm("sni")
	cert, err := a.serverService.GetNewEchCert(sni)
	if err != nil {
		jsonMsg(c, "get ech certificate", err)
		return
	}
	jsonObj(c, cert, nil)
}

// getNewVlessEnc generates a new VLESS encryption key.
// @Summary Generate VLESS encryption key
// @Description Generates a new VLESS encryption key
// @Tags Server
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/getNewVlessEnc [get]
func (a *ServerController) getNewVlessEnc(c *gin.Context) {
	out, err := a.serverService.GetNewVlessEnc()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.getNewVlessEncError"), err)
		return
	}
	jsonObj(c, out, nil)
}

// getNewUUID generates a new UUID.
// @Summary Generate UUID
// @Description Generates a new UUID v4
// @Tags Server
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/getNewUUID [get]
func (a *ServerController) getNewUUID(c *gin.Context) {
	uuidResp, err := a.serverService.GetNewUUID()
	if err != nil {
		jsonMsg(c, "Failed to generate UUID", err)
		return
	}

	jsonObj(c, uuidResp, nil)
}

// getNewmlkem768 generates a new ML-KEM-768 key.
// @Summary Generate ML-KEM-768 key
// @Description Generates a new ML-KEM-768 key pair
// @Tags Server
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/server/getNewmlkem768 [get]
func (a *ServerController) getNewmlkem768(c *gin.Context) {
	out, err := a.serverService.GetNewmlkem768()
	if err != nil {
		jsonMsg(c, "Failed to generate mlkem768 keys", err)
		return
	}
	jsonObj(c, out, nil)
}
