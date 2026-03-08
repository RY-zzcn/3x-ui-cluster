package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v2/web/service"
	"github.com/mhsanaei/3x-ui/v2/web/session"
)

type SlaveCertController struct {
	certService service.SlaveCertService
}

func NewSlaveCertController(g *gin.RouterGroup) *SlaveCertController {
	c := &SlaveCertController{}
	c.initRouter(g)
	return c
}

func (c *SlaveCertController) initRouter(g *gin.RouterGroup) {
	g.GET("/list", c.getAllCerts)
	g.GET("/slave/:slaveId", c.getCertsForSlave)
	g.POST("/del/:id", c.deleteCert)
}

// getAllCerts retrieves all slave certificates.
// @Summary List all certificates
// @Description Returns all TLS certificates across all slaves
// @Tags SlaveCerts
// @Produce json
// @Success 200 {object} entity.Msg
// @Router /panel/api/slave-certs/list [get]
func (c *SlaveCertController) getAllCerts(ctx *gin.Context) {
	if !session.IsLogin(ctx) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": "unauthorized"})
		return
	}

	certs, err := c.certService.GetAllCerts()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true, "obj": certs})
}

// getCertsForSlave retrieves certificates for a specific slave.
// @Summary Get certificates for slave
// @Description Returns TLS certificates for a specific slave node
// @Tags SlaveCerts
// @Produce json
// @Param slaveId path int true "Slave ID"
// @Success 200 {object} entity.Msg
// @Router /panel/api/slave-certs/slave/{slaveId} [get]
func (c *SlaveCertController) getCertsForSlave(ctx *gin.Context) {
	if !session.IsLogin(ctx) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": "unauthorized"})
		return
	}

	slaveId, err := strconv.Atoi(ctx.Param("slaveId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "invalid slave ID"})
		return
	}

	certs, err := c.certService.GetCertsForSlave(slaveId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true, "obj": certs})
}

// deleteCert deletes a slave certificate.
// @Summary Delete certificate
// @Description Deletes a TLS certificate by ID
// @Tags SlaveCerts
// @Produce json
// @Param id path int true "Certificate ID"
// @Success 200 {object} entity.Msg
// @Router /panel/api/slave-certs/del/{id} [post]
func (c *SlaveCertController) deleteCert(ctx *gin.Context) {
	if !session.IsLogin(ctx) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": "unauthorized"})
		return
	}

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "invalid cert ID"})
		return
	}

	if err := c.certService.DeleteCert(id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true, "msg": "Certificate deleted"})
}
