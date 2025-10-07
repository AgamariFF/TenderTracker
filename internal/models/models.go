package models

import "github.com/gin-gonic/gin"

type AllTenders struct {
	Vent []Tender
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
	VentCustomerPlace []string `form:"vent_customer_place"`
	VentDelKladrIds   []string `form:"vent_del_kladr_ids"`
}

func (c *Config) Bind(ctx *gin.Context) error {
	// Обрабатываем чекбоксы
	if ctx.PostForm("search_vent") == "on" {
		c.SearchVent = true
	} else {
		c.SearchVent = false
	}

	// Обрабатываем массивы
	c.VentCustomerPlace = ctx.PostFormArray("vent_customer_place")
	c.VentDelKladrIds = ctx.PostFormArray("vent_del_kladr_ids")

	return nil
}
