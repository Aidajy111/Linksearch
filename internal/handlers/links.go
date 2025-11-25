// бработчик /links
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

var proxy string = ""

type ReqLink struct { // структура для ответа
	Links    map[string]string `json:"links"`
	LinksNum int               `json:"links_num"`
}

// Links - структура для хранения непроверенных ссылок
type Links struct {
	Links []string
}

type Link struct {
	Status    string    `json:"status"`
	Items     ReqLink   `json:"items"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var metaText = []byte(`
	{
  		"last_links_num": 0
	}
`)

type MetaData struct {
	LastLinksNum int `json:"last_links_num"`
}

func FindLastLinkNum(filePath string) (int, error) {
	if _, err := os.Stat(filePath); err != nil {
		// Создание директории и файла
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return 0, fmt.Errorf("Directory creation error: %v", err)
		}

		file, err := os.Create(filePath)
		if err != nil {
			return 0, fmt.Errorf("Error when creating the file:", err)
		}
		defer file.Close()

		file.Write(metaText)

		fmt.Println("File success create:", filePath)

		return 0, nil
	} else {
		// обновление счетчика
		body, err := os.ReadFile(filePath)
		var meta MetaData
		if err != nil {
			return 0, fmt.Errorf("Error read file: %v\n", err)
		}

		if len(body) == 0 {
			return 0, fmt.Errorf("Body %s empty\n", body)
		}
		errs := json.Unmarshal(body, &meta)
		if errs != nil {
			return 0, fmt.Errorf("Parsing error JSON: %v", err)
		}
		oldValue := meta.LastLinksNum
		meta.LastLinksNum++

		newData, err := json.MarshalIndent(meta, "", "		")
		if err != nil {
			return 0, fmt.Errorf("Ошибка маршалинга JSON: %v", err)
		}

		e := os.WriteFile(filePath, newData, 0644)
		if e != nil {
			return 0, fmt.Errorf("Error writing to a file: %v", err)
		}

		fmt.Printf("Счетчик обновлен: %d -> %d\n", oldValue, meta.LastLinksNum)

		return meta.LastLinksNum, nil
	}
}

// Check - метод для работы с сылками
func (l *Links) Check() (ReqLink, error) {
	var req ReqLink
	req.Links = make(map[string]string)

	// получаем id для последней ссылки
	count, e := FindLastLinkNum("data/meta.json")
	if e != nil {
		fmt.Println(e)
	}

	req.LinksNum = count

	for _, link := range l.Links {
		if CheckLink("https://" + link) {
			req.Links[link] = "available"
		} else {
			req.Links[link] = "not available"
		}
	}

	err := SaveRequests("data/batches/", req)
	if err != nil {
		fmt.Println(e)
	}

	return req, nil
}

// SaveRequests - сохранить
func SaveRequests(directory string, d ReqLink) error {
	var l Link

	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("Error create directory: %v", err)
	}

	pathCreateFile := filepath.Join(directory, fmt.Sprintf("link_%d.json", d.LinksNum))

	l.Status = "done"
	l.Items = d
	l.CreatedAt = time.Now()
	l.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(l, "", "	")
	if err != nil {
		return fmt.Errorf("Marshalling error JSON: %v", err)
	}

	if err := os.WriteFile(pathCreateFile, data, 0644); err != nil {
		return fmt.Errorf("Error writing to a file: %v", err)
	}

	return nil
}

// CheckLink - если ссылка активная true, если нет понятно что))
func CheckLink(link string) bool {
	if link == "" {
		fmt.Println("The link is empty")
		return false
	}

	if proxy != "" {
		fmt.Println("use proxy: ", proxy)
	}

	client := http.Client{}

	if proxy != "" {
		proxyUrl, _ := url.Parse(proxy)
		client = http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	}
	req, _ := http.NewRequest("GET", link, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT x.y; Win64; x64; rv:10.0) Gecko/20100101 Firefox/10.0")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error: ", err)
		return false
	}
	_, errs := io.ReadAll(resp.Body)
	if errs == nil {
		if resp.StatusCode == 200 {
			fmt.Println(link + ": " + "ok")
			return true
		} else {
			fmt.Println("Site ", link, " returned error code: ", resp.StatusCode)
			return false
		}
	}
	defer resp.Body.Close()
	return true
}

// CreateLinksHandler - читает тело чекает ссылку и отвечает
func CreateLinksHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var links Links

	err = json.Unmarshal(body, &links)

	if err != nil {
		log.Fatal("Error during parsing:", err)
	}

	req, err := links.Check()
	if err != nil {
		log.Fatal("Error checking links:", err)
	}

	data, err := json.Marshal(req)
	if err != nil {
		fmt.Println(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
