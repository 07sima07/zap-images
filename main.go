package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/briandowns/spinner"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var dbName string
var dbUser string
var dbPass string
var dbServer string
var directory string
var threads int
var db *gorm.DB

func main() {
	input()
	db = openDb()

	println()
	fmt.Println("Load images data to RAM...")
	var groups []GroupParts
	db.Find(&groups)
	groupsLenDelimiter := len(groups) / threads

	println()
	fmt.Println("Loading images to disk ")
	s := spinner.New(spinner.CharSets[43], 100*time.Millisecond)
	s.Start()

	for i := 0; i < threads; i++ {
		if i+1 == threads {
			start := groupsLenDelimiter * i
			go imagesLoad(groups[start:])
		} else {
			start := groupsLenDelimiter * i
			end := groupsLenDelimiter + start
			go imagesLoad(groups[start:end])
		}
	}

	fmt.Scanf("h")
	s.Stop()
}

// work with images
func imagesLoad(groups []GroupParts) {
	for _, group := range groups {
		rawImages := imageColumnFormat(group)
		images := strings.Split(rawImages, ",")

		for i, img := range images {
			path := strings.Split(img, "/oem")[1]
			err := DownloadFile(directory+path, img)
			if err != nil {
				continue
			}

			if i == 1 {
				group.DownloadedImage2 = directory + path
			} else {
				group.DownloadedImage = directory + path
			}
		}

		if group.DownloadedImage == "" {
			continue
		}


		// update db
		db.Model(&GroupParts{}).Where("id = ?", group.ID).Update("downloaded_image", group.DownloadedImage)

		if group.DownloadedImage2 != "" {
			db.Model(&GroupParts{}).Where("id = ?", group.ID).Update("downloaded_image2", group.DownloadedImage2)
		}
	}

	fmt.Println("Thread downloaded all images")
}

// download file by url
func DownloadFile(filepath string, url string) error {
	if _, err := os.Stat(filepath); os.IsExist(err) {
		return nil
	}

	re := regexp.MustCompile(".*\\/")
	path := re.FindString(filepath)
	os.MkdirAll(path, os.ModePerm)

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

// format images raw data from database
func imageColumnFormat(parts GroupParts) string {
	raw := parts.Image.Value()

	raw = strings.ReplaceAll(raw, "[", "")
	raw = strings.ReplaceAll(raw, "]", "")
	raw = strings.ReplaceAll(raw, "\"\"", ",")
	raw = strings.ReplaceAll(raw, "\"", "")
	raw = strings.ReplaceAll(raw, "\\", "")

	return raw
}

// open database connection
func openDb() *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbServer, dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&GroupParts{})
	return db
}

// user input db params
func input() {
	fmt.Println("database name: ")
	fmt.Scanln(&dbName)
	if dbName == "" {
		dbName = "dacia"
	}

	fmt.Println("db user: ")
	fmt.Scanln(&dbUser)
	if dbUser == "" {
		dbUser = "max"
	}

	fmt.Println("db password: ")
	fmt.Scanln(&dbPass)
	if dbPass == "" {
		dbPass = "1"
	}

	fmt.Println("db server (default: localhost): ")
	fmt.Scanln(&dbServer)
	if dbServer == "" {
		dbServer = "localhost"
	}

	fmt.Println("directory for images (default: images): ")
	fmt.Scanln(&directory)
	if directory == "" {
		directory = "images"
	}

	fmt.Println("threads (default: 2): ")
	fmt.Scanln(&threads)
	if threads == 0 {
		threads = 2
	}

}

// gorm model group_part
type GroupParts struct {
	ID              uint   `gorm:"primarykey" json:"id"`
	Image           JSON   `sql:"type:json" json:"image"`
	DownloadedImage string `gorm:"downloaded_image"`
	DownloadedImage2 string `gorm:"downloaded_image2"`
}

// Mysql JSON support
type JSON []byte

func (j JSON) Value() string {
	if j.IsNull() {
		return ""
	}
	return string(j)
}
func (j *JSON) Scanln(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	s, ok := value.([]byte)
	if !ok {
		errors.New("Invalid Scanln Source")
	}
	*j = append((*j)[0:0], s...)
	return nil
}
func (m JSON) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}
func (m *JSON) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("null point exception")
	}
	*m = append((*m)[0:0], data...)
	return nil
}
func (j JSON) IsNull() bool {
	return len(j) == 0 || string(j) == "null"
}
func (j JSON) Equals(j1 JSON) bool {
	return bytes.Equal([]byte(j), []byte(j1))
}
