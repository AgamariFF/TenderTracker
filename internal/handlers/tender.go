package handlers

import (
	"net/http"
	"regexp"

	"tendertracker/internal/excel"
	"tendertracker/internal/logger"
	"tendertracker/internal/models"
	"tendertracker/internal/parsergovru"

	"github.com/gin-gonic/gin"
)

func searchTenders(re *regexp.Regexp) gin.HandlerFunc {
	return func(c *gin.Context) {
		allTenders := &models.AllTenders{}
		config := &models.Config{}

		if err := config.Bind(c); err != nil {
			logger.SugaredLogger.Warnf(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid input data",
				"details": err.Error(),
			})
			return
		}

		stats := map[string]interface{}{
			"totalFound": 0,
		}

		if config.SearchVent {
			tenders := parsergovru.ParseGovRu("vent", config, re)
			allTenders.Vent = tenders
			stats["totalFound"] = len(tenders)
		}

		file, err := excel.ToExcel(*config, allTenders)
		if err != nil {
			logger.SugaredLogger.Warnf(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to create Excel file",
				"details": err.Error(),
			})
			return
		}

		if err := file.SaveAs("Закупки.xlsx"); err != nil {
			logger.SugaredLogger.Warnf(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to save Excel file",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Excel file created successfully",
			"stats":    stats,
			"filename": "Закупки.xlsx",
		})
	}
}
