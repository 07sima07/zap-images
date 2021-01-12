package main

import (
	"fmt"
	"github.com/briandowns/spinner"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	fmt.Println("Alter table. Wait pls")
	db.Exec("alter table group_parts modify image varchar(500) not null")
	db.Exec("update group_parts set downloaded_image = null")
	fmt.Println("Load images data to RAM...")
	var groups []GroupParts
	db.Distinct("image").Find(&groups)

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

	k := false
	for k == false {
		var count int64
		db.Model(&GroupParts{}).Where("downloaded_image", nil).Count(&count)
		if count == 0 {
			k = true
		}
		time.Sleep(3 * time.Second)
	}

	s.Stop()

	fmt.Println("Well done")
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
		db.Model(&GroupParts{}).Where("image = ?", group.Image).Update("downloaded_image", group.DownloadedImage)

		if group.DownloadedImage2 != "" {
			db.Model(&GroupParts{}).Where("image = ?", group.Image).Update("downloaded_image2", group.DownloadedImage2)
		}
	}
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
	raw := parts.Image

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
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Discard})
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
	ID               uint   `gorm:"primarykey" json:"id"`
	Image            string   `json:"image"`
	DownloadedImage  string `gorm:"downloaded_image"`
	DownloadedImage2 string `gorm:"downloaded_image2"`
}