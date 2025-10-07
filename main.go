package main

import (
	"os"
	"regexp"
	"strings"
	"tendertracker/internal/handlers"
	"tendertracker/internal/logger"
)

func main() {
	logger.InitLogger("info")
	defer logger.Close()

	re, err := loadFilterPatterns("filter_patterns_vent.txt")
	if err != nil {
		logger.SugaredLogger.Errorf(err.Error())
	}

	router := handlers.SetupRouter(re)

	if err := router.Run(":8081"); err != nil {
		logger.SugaredLogger.Errorf(err.Error())
	}

}

func loadFilterPatterns(filename string) (*regexp.Regexp, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	pattern := strings.TrimSpace(string(data))

	return regexp.MustCompile(pattern), nil
}
