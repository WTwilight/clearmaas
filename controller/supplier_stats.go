package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// GET /api/supplier-stats/overview
func GetSupplierStatsOverview(c *gin.Context) {
	startTs, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTs, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	filter := service.SupplierStatsFilter{
		StartTimestamp: startTs,
		EndTimestamp:   endTs,
	}
	result, err := service.GetSupplierStatsOverview(filter)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": result})
}

// GET /api/supplier-stats/by-supplier
func GetSupplierStatsBySupplier(c *gin.Context) {
	startTs, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTs, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	filter := service.SupplierStatsFilter{
		StartTimestamp: startTs,
		EndTimestamp:   endTs,
	}
	result, err := service.GetSupplierStatsBySupplier(filter)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": result})
}

// GET /api/supplier-stats/by-channel
func GetSupplierStatsByChannel(c *gin.Context) {
	startTs, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTs, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	filter := service.SupplierStatsFilter{
		StartTimestamp: startTs,
		EndTimestamp:   endTs,
	}
	result, err := service.GetSupplierStatsByChannel(filter)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": result})
}

// GET /api/supplier-stats/by-model
func GetSupplierStatsByModel(c *gin.Context) {
	startTs, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTs, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	filter := service.SupplierStatsFilter{
		StartTimestamp: startTs,
		EndTimestamp:   endTs,
	}
	result, err := service.GetSupplierStatsByModel(filter)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": result})
}
