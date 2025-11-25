package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type LinkFile struct {
	Status    string    `json:"status"`
	Items     LinkItems `json:"items"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LinkItems struct {
	Links    map[string]string `json:"links"`
	LinksNum int               `json:"links_num"`
}

type LinkReport struct {
	URL      string
	Status   string
	SourceID int
}

// CollectLinks читает существующие JSON файлы и собирает все ссылки
func CollectLinks(ids []int) ([]LinkReport, error) {
	var allLinks []LinkReport
	basePath := "data/batches"

	for _, id := range ids {
		filename := fmt.Sprintf("link_%d.json", id)
		filePath := filepath.Join(basePath, filename)

		data, err := os.ReadFile(filePath)
		if err != nil {
			// Если файл не существует, пропускаем
			if os.IsNotExist(err) {
				fmt.Printf("File %s not found, skipping\n", filename)
				continue
			}
			return nil, fmt.Errorf("error reading file %s: %v", filename, err)
		}

		var linkFile LinkFile
		if err := json.Unmarshal(data, &linkFile); err != nil {
			return nil, fmt.Errorf("error parsing JSON from %s: %v", filename, err)
		}

		for url, status := range linkFile.Items.Links {
			allLinks = append(allLinks, LinkReport{
				URL:      url,
				Status:   status,
				SourceID: id,
			})
		}
	}

	return allLinks, nil
}

// generatePDF создает PDF отчет из собранных ссылок
func generatePDF(links []LinkReport) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	pdfFilename := fmt.Sprintf("report_%s.pdf", timestamp)

	tmpDir := os.TempDir()
	pdfPath := filepath.Join(tmpDir, pdfFilename)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// заголовок
	pdf.SetFont("Arial", "", 14)
	pdf.Cell(0, 8, "Report on the status of Internet links")
	pdf.Ln(10)

	// Шапка таблицы
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(20, 7, "ID", "1", 0, "L", false, 0, "")
	pdf.CellFormat(120, 7, "Link", "1", 0, "L", false, 0, "")
	pdf.CellFormat(30, 7, "Status", "1", 0, "L", false, 0, "")
	pdf.Ln(-1)

	// Строки таблицы
	pdf.SetFont("Arial", "", 9)
	for _, link := range links {
		url := link.URL
		if len(url) > 80 {
			url = url[:77] + "..."
		}

		pdf.CellFormat(20, 6, strconv.Itoa(link.SourceID), "1", 0, "L", false, 0, "")
		pdf.CellFormat(120, 6, url, "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, 6, link.Status, "1", 0, "L", false, 0, "")
		pdf.Ln(-1)
	}

	if err := pdf.OutputFileAndClose(pdfPath); err != nil {
		return "", fmt.Errorf("error saving PDF: %v", err)
	}

	return pdfPath, nil
}

// cleanupPDF удаляет временный PDF файл через указанное время
func cleanupPDF(filePath string, delay time.Duration) {
	time.Sleep(delay)

	if err := os.Remove(filePath); err != nil {
		fmt.Printf("Error deleting PDF file %s: %v\n", filePath, err)
	} else {
		fmt.Printf("PDF file %s deleted successfully\n", filePath)
	}
}

// ReportHandler - хендлер для генерации PDF отчетов
func ReportHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var requestData struct {
		LinksList []int `json:"links_list"`
	}

	if err := json.Unmarshal(body, &requestData); err != nil {
		http.Error(w, "Invalid JSON format: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(requestData.LinksList) == 0 {
		http.Error(w, "links_lists cannot be empty", http.StatusBadRequest)
		return
	}

	// Собираем ссылки из существующих файлов
	links, err := CollectLinks(requestData.LinksList)
	if err != nil {
		http.Error(w, "Error collecting links: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Если нет ссылок
	if len(links) == 0 {
		http.Error(w, "No links found for the provided IDs", http.StatusNotFound)
		return
	}

	// Генерируем PDF
	pdfPath, err := generatePDF(links)
	if err != nil {
		http.Error(w, "Error generating PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		http.Error(w, "Error reading PDF file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=link_report.pdf")
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfData)))

	// Отправляем PDF
	w.WriteHeader(http.StatusOK)
	w.Write(pdfData)

	// для удаления файла через время
	go cleanupPDF(pdfPath, 5*time.Minute)
}
