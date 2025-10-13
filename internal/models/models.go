package models

import (
	"github.com/gin-gonic/gin"
)

type AllTenders struct {
	Vent  []Tender
	Doors []Tender
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

	// Обрабатываем массивы
	c.VentCustomerPlace = ctx.PostFormArray("vent_customer_place")
	c.VentDelKladrIds = ctx.PostFormArray("vent_del_kladr_ids")

	return nil
}
