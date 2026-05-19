package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetSupplierStatsRouter(router *gin.Engine) {
	statsGroup := router.Group("/api/supplier-stats")
	statsGroup.Use(middleware.AdminAuth())
	{
		statsGroup.GET("/overview", controller.GetSupplierStatsOverview)
		statsGroup.GET("/by-supplier", controller.GetSupplierStatsBySupplier)
		statsGroup.GET("/by-channel", controller.GetSupplierStatsByChannel)
		statsGroup.GET("/by-model", controller.GetSupplierStatsByModel)
	}
}
