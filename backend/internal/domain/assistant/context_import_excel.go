package assistant

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type xlsxSharedStrings struct {
	Items []struct {
		Text string `xml:"t"`
	} `xml:"si"`
}

type xlsxWorksheet struct {
	Rows []struct {
		Cells []struct {
			Ref   string `xml:"r,attr"`
			Type  string `xml:"t,attr"`
			Value string `xml:"v"`
		} `xml:"c"`
	} `xml:"sheetData>row"`
}

func normalizeXLSXDictionary(source DictionaryImportSource) (DictionaryImportModel, error) {
	reader, err := zip.NewReader(bytes.NewReader(source.Content), int64(len(source.Content)))
	if err != nil {
		return DictionaryImportModel{}, fmt.Errorf("%w: invalid xlsx archive", ErrValidation)
	}
	shared, err := readXLSXSharedStrings(reader)
	if err != nil {
		return DictionaryImportModel{}, err
	}
	rows, err := readXLSXFirstSheet(reader, shared)
	if err != nil {
		return DictionaryImportModel{}, err
	}
	var csvText strings.Builder
	for rowIndex, row := range rows {
		if rowIndex > 0 {
			csvText.WriteByte('\n')
		}
		for colIndex, cell := range row {
			if colIndex > 0 {
				csvText.WriteByte(',')
			}
			csvText.WriteString(escapeCSVCell(cell))
		}
	}
	source.SourceType = ContextSourceCSV
	source.Content = []byte(csvText.String())
	model, err := normalizeCSVDictionary(source)
	model.SourceType = ContextSourceXLSX
	return model, err
}

func readXLSXSharedStrings(reader *zip.Reader) ([]string, error) {
	file := findZipFile(reader, "xl/sharedStrings.xml")
	if file == nil {
		return nil, nil
	}
	data, err := readZipFile(file)
	if err != nil {
		return nil, fmt.Errorf("%w: read shared strings", ErrValidation)
	}
	var parsed xlsxSharedStrings
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("%w: parse shared strings", ErrValidation)
	}
	values := make([]string, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		values = append(values, item.Text)
	}
	return values, nil
}

func readXLSXFirstSheet(reader *zip.Reader, shared []string) ([][]string, error) {
	file := findZipFile(reader, "xl/worksheets/sheet1.xml")
	if file == nil {
		return nil, fmt.Errorf("%w: xlsx sheet1.xml is required", ErrValidation)
	}
	data, err := readZipFile(file)
	if err != nil {
		return nil, fmt.Errorf("%w: read xlsx sheet", ErrValidation)
	}
	var parsed xlsxWorksheet
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("%w: parse xlsx sheet", ErrValidation)
	}
	rows := [][]string{}
	for _, row := range parsed.Rows {
		cells := []string{}
		for _, cell := range row.Cells {
			value := cell.Value
			if cell.Type == "s" {
				index, ok := parseSmallInt(value)
				if ok && index >= 0 && index < len(shared) {
					value = shared[index]
				}
			}
			cells = append(cells, value)
		}
		rows = append(rows, cells)
	}
	return rows, nil
}

func findZipFile(reader *zip.Reader, name string) *zip.File {
	for _, file := range reader.File {
		if file.Name == name {
			return file
		}
	}
	return nil
}

func readZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func escapeCSVCell(value string) string {
	if strings.ContainsAny(value, ",\"\n\r") {
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}
	return value
}

func parseSmallInt(value string) (int, bool) {
	total := 0
	if value == "" {
		return 0, false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, false
		}
		total = total*10 + int(ch-'0')
	}
	return total, true
}
