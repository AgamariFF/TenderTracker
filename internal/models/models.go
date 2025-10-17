package models

import (
	"strconv"
	"tendertracker/internal/logger"

	"github.com/gin-gonic/gin"
)

type AllTenders struct {
	Vent  []Tender
	Doors []Tender
	Build []Tender
	Metal []Tender
}

type Tender struct {
	Title       string
	Customer    string
	Price       string
	PublishDate string
	EndDate     string
	Link        string
}

type Config struct {
	SearchVent        bool     `form:"search_vent"`
	SearchDoors       bool     `form:"search_doors"`
	SearchBuild       bool     `form:"search_build"`
	SearchMetal       bool     `form:"search_metal`
	MinPriceVent      int      `form:"min_price_vent"`
	MinPriceDoors     int      `form:"min_price_doors"`
	MinPriceBuild     int      `form:"min_price_build"`
	MinPriceMetal     int      `form:"min_price_metal`
	VentCustomerPlace []string `form:"vent_customer_place"`
	VentDelKladrIds   []string `form:"vent_del_kladr_ids"`
}

func (c *Config) Bind(ctx *gin.Context) error {
	// Обрабатываем чекбоксы
	if ctx.PostForm("search_vent") == "on" || ctx.PostForm("search_vent") == "true" {
		c.SearchVent = true
	} else {
		c.SearchVent = false
	}

	if ctx.PostForm("search_doors") == "on" || ctx.PostForm("search_doors") == "true" {
		c.SearchDoors = true
	} else {
		c.SearchDoors = false
	}

	if ctx.PostForm("search_build") == "on" || ctx.PostForm("search_build") == "true" {
		c.SearchBuild = true
	} else {
		c.SearchBuild = false
	}

	if ctx.PostForm("search_metal") == "on" || ctx.PostForm("search_metal") == "true" {
		c.SearchMetal = true
	} else {
		c.SearchMetal = false
	}

	// Обрабатываем инты
	var err error
	if c.SearchVent {
		c.MinPriceVent, err = strconv.Atoi(ctx.PostForm("min_price_vent"))
		if err != nil {
			logger.SugaredLogger.Warnf(err.Error())
			c.MinPriceVent = 0
		}
	}
	if c.SearchDoors {
		c.MinPriceDoors, err = strconv.Atoi(ctx.PostForm("min_price_doors"))
		if err != nil {
			logger.SugaredLogger.Warnf(err.Error())
			c.MinPriceDoors = 0
		}
	}
	if c.SearchBuild {
		c.MinPriceBuild, err = strconv.Atoi(ctx.PostForm("min_price_build"))
		if err != nil {
			logger.SugaredLogger.Warnf(err.Error())
			c.MinPriceBuild = 0
		}
	}
	if c.SearchMetal {
		c.MinPriceMetal, err = strconv.Atoi(ctx.PostForm("min_price_metal"))
		if err != nil {
			logger.SugaredLogger.Warnf(err.Error())
			c.MinPriceMetal = 0
		}
	}

	// Обрабатываем массивы
	c.VentCustomerPlace = ctx.PostFormArray("vent_customer_place")
	c.VentDelKladrIds = ctx.PostFormArray("vent_del_kladr_ids")

	return nil
}
