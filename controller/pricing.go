package controller

import (
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func GetPricing(c *gin.Context) {
	userId, exists := c.Get("id")
	userID := 0
	if exists {
		userID = userId.(int)
	}
	pricingData := service.GetEffectivePricingForUser(userID, exists)

	c.JSON(200, gin.H{
		"success":            true,
		"data":               pricingData.Pricing,
		"vendors":            model.GetVendors(),
		"group_ratio":        pricingData.GroupRatio,
		"usable_group":       pricingData.UsableGroup,
		"supported_endpoint": model.GetSupportedEndpointMap(),
		"auto_groups":        pricingData.AutoGroups,
		"pricing_version":    "a42d372ccf0b5dd13ecf71203521f9d2",
		"ratio_source":       pricingData.RatioSource,
	})
}

func ResetModelRatio(c *gin.Context) {
	defaultStr := ratio_setting.DefaultModelRatio2JSONString()
	err := model.UpdateOption("ModelRatio", defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = ratio_setting.UpdateModelRatioByJSONString(defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "重置模型倍率成功",
	})
}
