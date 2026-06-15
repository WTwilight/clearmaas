package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetModelSquare(c *gin.Context) {
	userID := 0
	hasUser := false
	if id, exists := c.Get("id"); exists {
		if value, ok := id.(int); ok {
			userID = value
			hasUser = true
		}
	}
	data, err := service.GetModelSquareData(service.ModelSquareQuery{
		Keyword:  c.Query("keyword"),
		VendorID: c.Query("vendor_id"),
		Tag:      c.Query("tag"),
		UserID:   userID,
		HasUser:  hasUser,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ModelSquareResponse{
		Success: true,
		Data:    data,
	})
}

func GetModelSquareDetail(c *gin.Context) {
	userID := 0
	hasUser := false
	if id, exists := c.Get("id"); exists {
		if value, ok := id.(int); ok {
			userID = value
			hasUser = true
		}
	}
	data, err := service.GetModelSquareDetail(c.Param("model_name"), userID, hasUser)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}
