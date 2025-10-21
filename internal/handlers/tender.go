package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"sync"

	"tendertracker/internal/excel"
	"tendertracker/internal/logger"
	"tendertracker/internal/models"
	"tendertracker/internal/parsergovru"

	"github.com/gin-gonic/gin"
)

type parseResult struct {
	name    string
	tenders []models.Tender
	err     error
}

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

		// Канал для сбора результатов
		resultChan := make(chan parseResult, 4)
		var wg sync.WaitGroup
		var errors []string

		if config.SearchVent {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tenders, err := parsergovru.ParseGovRu("vent", config, re)
				resultChan <- parseResult{name: "vent", tenders: tenders, err: err}
			}()
		}

		if config.SearchDoors {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tenders, err := parsergovru.ParseGovRu("doors", config, re)
				resultChan <- parseResult{name: "doors", tenders: tenders, err: err}
			}()
		}

		if config.SearchBuild {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tenders, err := parsergovru.ParseGovRu("build", config, re)
				resultChan <- parseResult{name: "build", tenders: tenders, err: err}
			}()
		}

		if config.SearchMetal {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tenders, err := parsergovru.ParseGovRu("metal", config, re)
				resultChan <- parseResult{name: "metal", tenders: tenders, err: err}
			}()
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		for result := range resultChan {
			if result.err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", result.name, result.err))
				continue
			}

			switch result.name {
			case "vent":
				allTenders.Vent = result.tenders
				stats["ventFound"] = len(result.tenders)
				stats["totalFound"] += len(result.tenders)
			case "doors":
				allTenders.Doors = result.tenders
				stats["doorsFound"] = len(result.tenders)
				stats["totalFound"] += len(result.tenders)
			case "build":
				allTenders.Build = result.tenders
				stats["buildFound"] = len(result.tenders)
				stats["totalFound"] += len(result.tenders)
			case "metal":
				allTenders.Metal = result.tenders
				stats["metalFound"] = len(result.tenders)
				stats["totalFound"] += len(result.tenders)
			}
		}

		if len(errors) > 0 && stats["totalFound"] == 0 {
			logger.SugaredLogger.Warnf("All parsing attempts failed: %v", errors)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to parse tenders from external source",
				"details": fmt.Sprintf("Errors: %v", errors),
				"stats":   stats,
			})
			return
		}

		if len(errors) > 0 {
			logger.SugaredLogger.Warnf("Partial parsing errors (but some tenders found): %v", errors)
		}

		if stats["totalFound"] == 0 {
			logger.SugaredLogger.Warn("0 tenders found")
			c.JSON(http.StatusNotFound, gin.H{
				"error": "No tenders found matching the criteria",
				"stats": stats,
			})
			return
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

		response := gin.H{
			"message":  "Excel file created successfully",
			"stats":    stats,
			"filename": "Закупки.xlsx",
		}

		if len(errors) > 0 {
			response["warnings"] = errors
		}

		c.JSON(http.StatusOK, response)
	}
}
