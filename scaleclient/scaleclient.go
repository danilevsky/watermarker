package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
)

func appendFileToWriter(filename string, field string, bodyWriter *multipart.Writer) error {

	fileWriter, err := bodyWriter.CreateFormFile(field, filename)
	if err != nil {
		fmt.Println("Error create form file. Filename:" + filename + " field:" + field)
		return err
	}

	fh, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Can not open file:%s\n", filename)
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		fmt.Printf("Error copy file: %s to writer, err:%s\n", filename, err)
		return err
	}

	return nil
}

func postFile(filename string, watermark string, targetURL string, outFilename string) error {

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	err := appendFileToWriter(filename, "image", bodyWriter)
	if err != nil {
		return err
	}

	err = appendFileToWriter(watermark, "watermark", bodyWriter)
	if err != nil {
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post(targetURL, contentType, bodyBuf)
	if err != nil {
		fmt.Println("Error post file")
		return err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error read response")
		return err
	}
	fmt.Println(resp.Status)

	err = ioutil.WriteFile(outFilename, respBody, 0644)
	if err != nil {
		fmt.Println("Error save response to file")
		return err
	}

	return nil
}

func main() {
	targetURL := "http://localhost:3210/watermark"

	basePtr := flag.String("base", "", "base image")
	watermarkPtr := flag.String("watermark", "", "watermark image")
	outFilenamePtr := flag.String("outfile", "result.png", "out file name (png)")
	flag.Parse()

	if *basePtr == "" || *watermarkPtr == "" {
		fmt.Println("Invalid arguments")
		flag.PrintDefaults()
		return
	}

	postFile(*basePtr, *watermarkPtr, targetURL, *outFilenamePtr)
}
