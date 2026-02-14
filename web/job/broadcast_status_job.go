package job

import (
	"github.com/mhsanaei/3x-ui/v2/logger"
	"github.com/mhsanaei/3x-ui/v2/web/service"
	"github.com/mhsanaei/3x-ui/v2/web/websocket"
)

// BroadcastStatusJob periodically broadcasts inbound/outbound status to frontend
// This ensures real-time updates even when slaves don't send traffic data
type BroadcastStatusJob struct {
	inboundService  service.InboundService
	outboundService service.OutboundService
}

// NewBroadcastStatusJob creates a new broadcast status job instance
func NewBroadcastStatusJob() *BroadcastStatusJob {
	return new(BroadcastStatusJob)
}

// Run broadcasts current inbound/outbound status from database to all connected clients
func (j *BroadcastStatusJob) Run() {
	// Fetch updated inbounds from database with accumulated traffic values
	updatedInbounds, err := j.inboundService.GetAllInbounds()
	if err != nil {
		logger.Debug("broadcast_status_job: failed to get inbounds:", err)
		return
	}

	updatedOutbounds, err := j.outboundService.GetOutboundsTraffic()
	if err != nil {
		logger.Debug("broadcast_status_job: failed to get outbounds:", err)
	}

	// Get online clients and last online map for real-time status updates
	onlineClients := j.inboundService.GetOnlineClients()
	lastOnlineMap, err := j.inboundService.GetClientsLastOnline()
	if err != nil {
		logger.Debug("broadcast_status_job: failed to get last online map:", err)
		lastOnlineMap = make(map[string]int64)
	}

	// Broadcast full inbounds update for real-time UI refresh
	if updatedInbounds != nil && len(updatedInbounds) > 0 {
		websocket.BroadcastInbounds(updatedInbounds)
		logger.Debug("broadcast_status_job: broadcasted inbounds update")
	}

	if updatedOutbounds != nil && len(updatedOutbounds) > 0 {
		websocket.BroadcastOutbounds(updatedOutbounds)
	}

	// Broadcast traffic update with online status
	trafficUpdate := map[string]any{
		"onlineClients": onlineClients,
		"lastOnlineMap": lastOnlineMap,
	}
	websocket.BroadcastTraffic(trafficUpdate)
	logger.Debugf("broadcast_status_job: broadcasted status (%d online clients)", len(onlineClients))
}
