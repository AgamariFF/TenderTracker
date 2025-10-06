package excel

import (
	"tendertracker/internal/models"
	"time"

	"github.com/xuri/excelize/v2"
)

func ToExcel(tenders *[]models.Tender, name string) (*excelize.File, error) {
	excelFile, err := excelCreat(name, nil)

	if err != nil {
		return nil, err
	}

	return excelFile, nil
}

func excelCreat(sheet string, f *excelize.File) (*excelize.File, error) {
	if f == nil {
		f = excelize.NewFile()
	}
	style, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal:     "center",
			Indent:         1,
			ReadingOrder:   0,
			RelativeIndent: 1,
			ShrinkToFit:    true,
			TextRotation:   0,
			Vertical:       "",
			WrapText:       true,
		},
		Font: &excelize.Font{
			Bold:      true,
			Italic:    false,
			Underline: "",
			Family:    "",
			Size:      14,
			Strike:    false,
		},
	})

	if err != nil {
		return nil, err
	}

	f.SetColWidth(sheet, "A", "B", 16)
	f.SetColWidth(sheet, "C", "C", 35)
	f.SetColWidth(sheet, "D", "E", 50)
	f.SetColWidth(sheet, "F", "F", 16)
	f.SetColWidth(sheet, "G", "G", 30)
	f.SetColWidth(sheet, "H", "H", 17)
	f.SetCellValue(sheet, "A1", "Дата размещения")
	f.SetCellValue(sheet, "B1", "Дата окончания")
	f.SetCellValue(sheet, "C1", "Заказчик кратко")
	f.SetCellValue(sheet, "E1", "Объект закупки + ссылка")
	f.SetCellValue(sheet, "F1", "Начальная цена")
	f.SetCellStyle(sheet, "A1", "F1", style)
	f.SetCellValue(sheet, "H1", "Дата создания таблицы: "+time.Now().UTC().Format("02.01.2006"))

	return f, nil
}
