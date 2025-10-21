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
	"tendertracker/internal/parsersber"

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
		allTenders := &models.TendersFromAllSites{}
		config := &models.Config{}

		if err := config.Bind(c); err != nil {
			logger.SugaredLogger.Warnf(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid input data",
				"details": err.Error(),
			})
			return
		}

		var statsZakupkiGovRu map[string]int
		var statsSber map[string]int
		var err error
		var errors []error

		logger.SugaredLogger.Infof("config: %+v", config)

		allTenders.ZakupkiGovRu, statsZakupkiGovRu, err = SearchFromZakupkigovru(re, config)
		if err != nil {
			logger.SugaredLogger.Warn(err)
			errors = append(errors, err)
		}
		allTenders.ZakupkiSber, statsSber, err = SearchFromSber(re, config)
		if err != nil {
			logger.SugaredLogger.Warn(err)
			errors = append(errors, err)
		}

		stats := mergeMaps(statsSber, statsZakupkiGovRu)
		stats["totalFound"] = statsZakupkiGovRu["totalFoundZakupkiGovRu"] + statsSber["totalFoundSber"]

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

func SearchFromZakupkigovru(re *regexp.Regexp, config *models.Config) (models.AllTenders, map[string]int, error) {
	var allTenders models.AllTenders
	stats := map[string]int{
		"totalFound": 0,
	}

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
			stats["ventFoundZakupkiGovRu"] = len(result.tenders)
			stats["totalFoundZakupkiGovRu"] += len(result.tenders)
		case "doors":
			allTenders.Doors = result.tenders
			stats["doorsFoundZakupkiGovRu"] = len(result.tenders)
			stats["totalFoundZakupkiGovRu"] += len(result.tenders)
		case "build":
			allTenders.Build = result.tenders
			stats["buildFoundZakupkiGovRu"] = len(result.tenders)
			stats["totalFoundZakupkiGovRu"] += len(result.tenders)
		case "metal":
			allTenders.Metal = result.tenders
			stats["metalFoundZakupkiGovRu"] = len(result.tenders)
			stats["totalFoundZakupkiGovRu"] += len(result.tenders)
		}
	}

	var err error

	if len(errors) > 0 && stats["totalFoundZakupkiGovRu"] == 0 {
		logger.SugaredLogger.Warnf("All parsing attempts failed from ZakupkiGovRu: %v", errors)
		return allTenders, stats, err
	}

	if len(errors) > 0 {
		logger.SugaredLogger.Warnf("Partial parsing errors (but some tenders found) from ZakupkiGovRu: %v", errors)
		err = fmt.Errorf("Error when parsing ZakupkiGovRu: %v", errors)
	}

	if stats["totalFoundZakupkiGovRu"] == 0 {
		logger.SugaredLogger.Warn("0 tenders found from ZakupkiGovRu")
		return allTenders, stats, err
	}

	return allTenders, stats, err
}

func SearchFromSber(re *regexp.Regexp, config *models.Config) (models.AllTenders, map[string]int, error) {
	var allTenders models.AllTenders
	stats := map[string]int{
		"totalFound": 0,
	}

	resultChan := make(chan parseResult, 4)
	var wg sync.WaitGroup
	var errors []string

	if config.SearchVent {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tenders, err := parsersber.ParseSberAst("vent", config, re)
			resultChan <- parseResult{name: "vent", tenders: tenders, err: err}
		}()
	}

	if config.SearchDoors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tenders, err := parsersber.ParseSberAst("doors", config, re)
			resultChan <- parseResult{name: "doors", tenders: tenders, err: err}
		}()
	}

	if config.SearchBuild {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tenders, err := parsersber.ParseSberAst("build", config, re)
			resultChan <- parseResult{name: "build", tenders: tenders, err: err}
		}()
	}

	if config.SearchMetal {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tenders, err := parsersber.ParseSberAst("metal", config, re)
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
			stats["ventFoundSber"] = len(result.tenders)
			stats["totalFoundSber"] += len(result.tenders)
		case "doors":
			allTenders.Doors = result.tenders
			stats["doorsFoundSber"] = len(result.tenders)
			stats["totalFoundSber"] += len(result.tenders)
		case "build":
			allTenders.Build = result.tenders
			stats["buildFoundSber"] = len(result.tenders)
			stats["totalFoundSber"] += len(result.tenders)
		case "metal":
			allTenders.Metal = result.tenders
			stats["metalFoundSber"] = len(result.tenders)
			stats["totalFoundSber"] += len(result.tenders)
		}
	}

	var err error

	if len(errors) > 0 && stats["totalFoundSber"] == 0 {
		logger.SugaredLogger.Warnf("All parsing attempts failed from Sber: %v", errors)
		return allTenders, stats, err
	}

	if len(errors) > 0 {
		logger.SugaredLogger.Warnf("Partial parsing errors (but some tenders found) from Sber: %v", errors)
		err = fmt.Errorf("Error when parsing Sber: %v", errors)
	}

	if stats["totalFoundSber"] == 0 {
		logger.SugaredLogger.Warn("0 tenders found from Sber")
		return allTenders, stats, err
	}

	return allTenders, stats, err
}

func mergeMaps(maps ...map[string]int) map[string]int {
	result := make(map[string]int)

	for _, m := range maps {
		for key, value := range m {
			result[key] += value
		}
	}

	return result
}
