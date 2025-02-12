package controller

import (
	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

type APIController struct {
	BaseController
	inboundController *InboundController
	outboundController *OutboundController
	routingController *RoutingController
	Tgbot             service.Tgbot
}

func NewAPIController(g *gin.RouterGroup) *APIController {
	a := &APIController{}
	a.initRouter(g)
	return a
}

func (a *APIController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/panel/api")
	g.Use(a.checkLogin)

	a.inboundController = NewInboundController(g.Group("/inbounds"))
	a.outboundController = NewOutboundController(g.Group("/outbounds"))
	a.routingController = NewRoutingController(g.Group("/routing"))

	inboundRoutes := []struct {
		Method  string
		Path    string
		Handler gin.HandlerFunc
	}{
		{"GET", "/createbackup", a.createBackup},
		{"GET", "/inbounds/list", a.inboundController.getInbounds},
		{"GET", "/inbounds/get/:id", a.inboundController.getInbound},
		{"GET", "/inbounds/getClientTraffics/:email", a.inboundController.getClientTraffics},
		{"GET", "/inbounds/getClientTrafficsById/:id", a.inboundController.getClientTrafficsById},
		{"POST", "/inbounds/add", a.inboundController.addInbound},
		{"POST", "/inbounds/del/:id", a.inboundController.delInbound},
		{"POST", "/inbounds/update/:id", a.inboundController.updateInbound},
		{"POST", "/inbounds/clientIps/:email", a.inboundController.getClientIps},
		{"POST", "/inbounds/clearClientIps/:email", a.inboundController.clearClientIps},
		{"POST", "/inbounds/addClient", a.inboundController.addInboundClient},
		{"POST", "/inbounds/:id/delClient/:clientId", a.inboundController.delInboundClient},
		{"POST", "/inbounds/updateClient/:clientId", a.inboundController.updateInboundClient},
		{"POST", "/inbounds/:id/resetClientTraffic/:email", a.inboundController.resetClientTraffic},
		{"POST", "/inbounds/resetAllTraffics", a.inboundController.resetAllTraffics},
		{"POST", "/inbounds/resetAllClientTraffics/:id", a.inboundController.resetAllClientTraffics},
		{"POST", "/inbounds/delDepletedClients/:id", a.inboundController.delDepletedClients},
		{"POST", "/inbounds/onlines", a.inboundController.onlines},
	}

	outboundRoutes := []struct {
		Method  string
		Path    string
		Handler gin.HandlerFunc
	}{
		// Add outbound routes here
	}

	routingRoutes := []struct {
		Method  string
		Path    string
		Handler gin.HandlerFunc
	}{
		// Add routing routes here
	}

	for _, route := range inboundRoutes {
		g.Handle(route.Method, route.Path, route.Handler)
	}

	for _, route := range outboundRoutes {
		g.Handle(route.Method, route.Path, route.Handler)
	}

	for _, route := range routingRoutes {
		g.Handle(route.Method, route.Path, route.Handler)
	}
}

func (a *APIController) createBackup(c *gin.Context) {
	a.Tgbot.SendBackupToAdmins()
}
