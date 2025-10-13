package excel

import (
	"strconv"
	"tendertracker/internal/models"
	"time"

	"github.com/xuri/excelize/v2"
)

func ToExcel(config models.Config, allTenders *models.AllTenders) (*excelize.File, error) {
	excelFile := excelize.NewFile()

	if config.SearchVent {
		addTendersAndSheet(excelFile, allTenders.Vent, "Вентиляция")
		addTendersAndSheet(excelFile, allTenders.Doors, "Двери")
		addTendersAndSheet(excelFile, allTenders.Build, "Строительство/Реконструкция")
	}

	return excelFile, nil
}

func addTendersAndSheet(f *excelize.File, tenders []models.Tender, sheet string) error {
	f.NewSheet(sheet)

	index, _ := f.GetSheetIndex("Sheet1")
	if index != -1 {
		f.DeleteSheet("Sheet1")
	}

	CreateHeader(f, sheet)

	for index, value := range tenders {
		f.SetCellValue(sheet, "A"+strconv.Itoa(index+2), value.PublishDate)
		f.SetCellValue(sheet, "B"+strconv.Itoa(index+2), value.EndDate)
		f.SetCellValue(sheet, "C"+strconv.Itoa(index+2), value.Customer)
		f.SetCellValue(sheet, "D"+strconv.Itoa(index+2), value.Title)
		f.SetCellValue(sheet, "E"+strconv.Itoa(index+2), value.Price)

		f.SetCellHyperLink(sheet, "D"+strconv.Itoa(index+2), value.Link, "External")
	}

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
	f.SetColWidth(sheet, "C", "C", 40)
	f.SetColWidth(sheet, "D", "D", 100)
	f.SetColWidth(sheet, "E", "E", 20)
	f.SetCellValue(sheet, "A1", "Дата размещения")
	f.SetCellValue(sheet, "B1", "Дата окончания")
	f.SetCellValue(sheet, "C1", "Заказчик")
	f.SetCellValue(sheet, "D1", "Объект закупки + ссылка")
	f.SetCellValue(sheet, "E1", "Начальная цена")
	f.SetCellStyle(sheet, "A1", "E1", style)
	f.SetCellValue(sheet, "F1", "Дата создания таблицы: "+time.Now().UTC().Format("02.01.2006"))

	return nil
}
