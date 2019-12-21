package main

import (
	"errors"
	"fmt"
	"github.com/nfnt/resize"
	"image"
	_ "image"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var myMap = make(map[string]bool)

func main() {

	http.HandleFunc("/thumbnail", resizeImage)
	http.HandleFunc("/", handleUnknown)
	if err := http.ListenAndServe(os.Getenv("PORT"), nil); err != nil {
		panic(err)
	}
}
func handleUnknown(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(404)
	_, _ = w.Write([]byte("Could not find a valid end point.\n"))
}

// /thumbnail?url=<url>&width=<width>&height=<height>
func resizeImage(w http.ResponseWriter, r *http.Request) {
	fullUrlFile, width, height, done := readQueryParams(r)
	if done {
		failGracefully(w, errors.New("missing query params"))
		return
	}
	fileName, done := downloadImage(w, fullUrlFile, width, height)
	if done {
		return
	}
	newFileName := renameOriginalImage(w, fileName)
	myMap[newFileName] = true

	wi, _ := strconv.ParseUint(width, 10, 0)
	he, _ := strconv.ParseUint(height, 10, 0)
	changeSize(w, fileName, newFileName, uint(wi), uint(he))
}

func readQueryParams(r *http.Request) (string, string, string ,bool) {
	queryParams := r.URL.Query()
	fullUrlFile := queryParams.Get("url")
	width := queryParams.Get("width")
	height := queryParams.Get("height")
	if fullUrlFile == "" || width == "" || height == "" {
		fmt.Println("missing query params")
		return "", "", "", true
	}
	return fullUrlFile, width, height, false
}

func downloadImage(w http.ResponseWriter, fullUrlFile string, width string, height string) (string, bool) {

	//create file with fileName, download content to the file
	fileName := buildFileName(w, fullUrlFile)

	if fileName == "" {
		return "", true
	}
	if haveInMap(fileName, width, height, w) {
		return fileName,true
	}

	file := createFile(w, fileName)
	putFile(w, file, httpClient(), fullUrlFile)

	return fileName, false
}

func haveInMap(fileName string, width string, height string, w http.ResponseWriter) bool {

	name := "w_" + width + "h_" + height + fileName
	val := myMap[name]
	if val {
		w.Header().Set("Content-Type", "image/jpeg") // <-- set the content-type header
		// decode jpeg into image.Image
		reader, err := os.Open(name)
		if err != nil {
			failGracefully(w, err)
			return true
		}
		defer reader.Close()
		img, err := jpeg.Decode(reader)
		if err != nil {
			failGracefully(w, err)
			return true
		}
		err = jpeg.Encode(w, img, nil)
		if err != nil {
			failGracefully(w, err)
			return true
		}
		return true
	}
	return false
}

func buildFileName(w http.ResponseWriter, fullUrlFile string) string {
	fileUrl, err := url.Parse(fullUrlFile)
	if err != nil {
		failGracefully(w, err)
		return ""
	}

	path := fileUrl.Path
	segments := strings.Split(path, "/")

	fileName := segments[len(segments)-1]
	ext := filepath.Ext(fileName)
	if ext != ".jpg" {
		failGracefully(w, errors.New("not a jpg"))
		return ""
	}

	return fileName
}

func createFile(w http.ResponseWriter, fileName string) *os.File {
	file, err := os.Create(fileName)
	if err != nil {
		failGracefully(w, err)
		return nil
	}
	return file
}

func changeSize(w http.ResponseWriter, fileName string, newFileName string, width uint, height uint) {

	// decode jpeg into image.Image
	img := readFileIntoImg(w, newFileName)
	if img == nil {
		return
	}

	m := resize.Thumbnail(width, height, img, resize.Lanczos3)

	resizedName := "w_" + strconv.Itoa(int(width)) + "h_" + strconv.Itoa(int(height)) + fileName
	out := createFile(w, resizedName)

	// write new image to file
	err := jpeg.Encode(out, m, nil)
	if err != nil {
		failGracefully(w, err)
		return
	}
	myMap[resizedName] = true

	w.Header().Set("Content-Type", "image/jpeg") // <-- set the content-type header
	err = jpeg.Encode(w, m, nil)
	if err != nil {
		failGracefully(w, err)
		return
	}
}

func readFileIntoImg(w http.ResponseWriter, newFileName string) image.Image {
	reader, err := os.Open(newFileName)
	if err != nil {
		failGracefully(w, err)
		return nil
	}
	defer reader.Close()
	img, err := jpeg.Decode(reader)
	if err != nil {
		failGracefully(w, err)
		return nil
	}
	return img
}

func renameOriginalImage(w http.ResponseWriter, fileName string) string {
	reader, err := os.Open(fileName)
	if err != nil {
		failGracefully(w, err)
		return ""
	}
	defer reader.Close()
	config, err := jpeg.DecodeConfig(reader)
	if err != nil {
		failGracefully(w, err)
		return ""
	}
	origH := config.Height
	origW := config.Width

	newFileName := "w_" + strconv.Itoa(origW) + "h_" + strconv.Itoa(origH) + fileName
	createFile(w, newFileName)
	//save with new name
	err = os.Rename(fileName, newFileName)
	if err != nil {
		failGracefully(w, err)
		return ""
	}
	return newFileName
}

func putFile(w http.ResponseWriter, file *os.File, client *http.Client, fullUrl string) {
	resp, err := client.Get(fullUrl)

	if err != nil {
		failGracefully(w, err)
		return
	}
	defer resp.Body.Close()

	size, err := io.Copy(file, resp.Body)
	if size<1 {
		failGracefully(w, err)
		return
	}

	defer file.Close()

	if err != nil {
		failGracefully(w, err)
		return
	}
}

func httpClient() *http.Client {
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	return &client
}

func failGracefully(w http.ResponseWriter, err error)  {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = fmt.Fprintf(w, `{"result":"","error":%q}`, err)
}
