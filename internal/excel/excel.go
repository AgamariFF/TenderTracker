package excel

import (
	"strconv"
	"tendertracker/internal/logger"
	"tendertracker/internal/models"
	"time"

	"github.com/xuri/excelize/v2"
)

func ToExcel(config models.Config, allTenders *models.TendersFromAllSites) (*excelize.File, error) {
	excelFile := excelize.NewFile()

	if config.SearchVent {
		if err := addTendersAndSheet(excelFile, allTenders.ZakupkiGovRu.Vent, allTenders.ZakupkiSber.Vent, "Вентиляция"); err != nil {
			logger.SugaredLogger.Warn(err)
		}
	}
	if config.SearchDoors {
		if err := addTendersAndSheet(excelFile, allTenders.ZakupkiGovRu.Doors, allTenders.ZakupkiSber.Doors, "Двери"); err != nil {
			logger.SugaredLogger.Warn(err)
		}
	}
	if config.SearchBuild {
		if err := addTendersAndSheet(excelFile, allTenders.ZakupkiGovRu.Build, allTenders.ZakupkiSber.Build, "Строительство"); err != nil {
			logger.SugaredLogger.Warn(err)
		}
	}
	if config.SearchMetal {
		if err := addTendersAndSheet(excelFile, allTenders.ZakupkiGovRu.Metal, allTenders.ZakupkiSber.Metal, "Металл."); err != nil {
			logger.SugaredLogger.Warn(err)
		}
	}

	return excelFile, nil
}

func addTendersAndSheet(f *excelize.File, tendersZakupkiGovRu, tendersSber []models.Tender, sheet string) error {
	f.NewSheet(sheet)

	index, _ := f.GetSheetIndex("Sheet1")
	if index != -1 {
		f.DeleteSheet("Sheet1")
	}

	CreateHeader(f, sheet)

	titleStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 18,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return err
	}

	err = f.MergeCell(sheet, "A2", "E2")
	if err != nil {
		return err
	}

	err = f.SetCellStyle(sheet, "A2", "E2", titleStyle)
	if err != nil {
		return err
	}

	f.SetCellValue(sheet, "A2", "Zakupki.Gov.ru")

	index = 3

	setTenderInf(f, sheet, tendersZakupkiGovRu, &index)

	err = f.MergeCell(sheet, "A"+strconv.Itoa(index), "E"+strconv.Itoa(index))
	if err != nil {
		return err
	}

	err = f.SetCellStyle(sheet, "A"+strconv.Itoa(index), "E"+strconv.Itoa(index), titleStyle)
	if err != nil {
		return err
	}

	f.SetCellValue(sheet, "A"+strconv.Itoa(index), "Сбер-АСТ")
	logger.SugaredLogger.Debugf("Added title Sber to sheet: %s to line: %s", sheet, index)

	index++

	setTenderInf(f, sheet, tendersSber, &index)

	return nil
}

func CreateHeader(f *excelize.File, sheet string) error {
	style, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal:     "center",
			Indent:         1,
			ReadingOrder:   0,
			RelativeIndent: 1,
			ShrinkToFit:    false,
			TextRotation:   0,
			Vertical:       "",
			WrapText:       true,
		},
		Font: &excelize.Font{
			Bold:      true,
			Italic:    false,
			Underline: "",
			Family:    "",
			Size:      12,
			Strike:    false,
		},
	})

	if err != nil {
		return err
	}

	f.SetColWidth(sheet, "A", "B", 16)
	f.SetColWidth(sheet, "C", "C", 34)
	f.SetColWidth(sheet, "D", "D", 40)
	f.SetColWidth(sheet, "E", "E", 100)
	f.SetColWidth(sheet, "F", "F", 20)
	f.SetCellValue(sheet, "A1", "Дата размещения")
	f.SetCellValue(sheet, "B1", "Дата окончания")
	f.SetCellValue(sheet, "C1", "Расположение")
	f.SetCellValue(sheet, "D1", "Заказчик")
	f.SetCellValue(sheet, "E1", "Объект закупки + ссылка")
	f.SetCellValue(sheet, "F1", "Начальная цена")
	f.SetCellStyle(sheet, "A1", "F1", style)
	f.SetCellValue(sheet, "G1", "Дата создания таблицы: "+time.Now().UTC().Format("02.01.2006"))

	return nil
}

func setTenderInf(f *excelize.File, sheet string, tender []models.Tender, index *int) {
	for _, value := range tender {
		f.SetCellValue(sheet, "A"+strconv.Itoa(*index), value.PublishDate)
		f.SetCellValue(sheet, "B"+strconv.Itoa(*index), value.EndDate)
		f.SetCellValue(sheet, "C"+strconv.Itoa(*index), value.Region)
		f.SetCellValue(sheet, "D"+strconv.Itoa(*index), value.Customer)
		f.SetCellValue(sheet, "E"+strconv.Itoa(*index), value.Title)
		f.SetCellValue(sheet, "F"+strconv.Itoa(*index), value.Price)

		f.SetCellHyperLink(sheet, "E"+strconv.Itoa(*index), value.Link, "External")
		*index++
	}
}
