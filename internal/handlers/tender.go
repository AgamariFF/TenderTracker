package handlers

import (
	"net/http"
	"regexp"
	"sync"

	"tendertracker/internal/excel"
	"tendertracker/internal/logger"
	"tendertracker/internal/models"
	"tendertracker/internal/parsergovru"

	"github.com/gin-gonic/gin"
)

func searchTenders(re *regexp.Regexp) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.SugaredLogger.Infof("Starting search tenders.")
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

		stats := map[string]int{
			"totalFound": 0,
		}

		logger.SugaredLogger.Infof("config: %+v", config)

		if config.SearchVent || config.SearchDoors {
			var wg sync.WaitGroup
			var mu sync.Mutex

			if config.SearchVent {
				wg.Add(1)
				go func() {
					defer wg.Done()
					tenders := parsergovru.ParseGovRu("vent", config, re)
					allTenders.Vent = tenders
					stats["ventFound"] = len(tenders)
					mu.Lock()
					stats["totalFound"] += stats["ventFound"]
					mu.Unlock()
				}()
			}

			if config.SearchDoors {
				wg.Add(1)
				go func() {
					defer wg.Done()
					tenders := parsergovru.ParseGovRu("doors", config, re)
					allTenders.Doors = tenders
					stats["doorsFound"] = len(tenders)
					mu.Lock()
					stats["totalFound"] += stats["doorsFound"]
					mu.Unlock()
				}()
			}

			if config.SearchBuild {
				wg.Add(1)
				go func() {
					defer wg.Done()
					tenders := parsergovru.ParseGovRu("build", config, re)
					allTenders.Build = tenders
					stats["buildFound"] = len(tenders)
					mu.Lock()
					stats["totalFound"] += stats["doorsFound"]
					mu.Unlock()
				}()
			}

			wg.Wait()

		}

		if len(allTenders.Doors)+len(allTenders.Vent) == 0 {
			logger.SugaredLogger.Warn("0 tenders found")
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
