package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"golang.org/x/image/draw"
)

//resize image with aspect ratio
func resize(src image.Image, width int, height int) image.Image {
	srcSize := src.Bounds().Size()
	if width == srcSize.Y && height == srcSize.X {
		return src
	}

	srcWidth := float32(srcSize.X)
	srcHeight := float32(srcSize.Y)

	aspectW := float32(float32(width) / float32(srcWidth))
	aspectH := float32(float32(height) / float32(srcHeight))
	aspect := float32(0)

	destX := 0
	destY := 0

	if aspectH < aspectW {
		aspect = aspectH
		destX = int(float32(width)/float32(2) - (srcWidth*aspect)/float32(2))
	} else {
		aspect = aspectW
		destY = int(float32(height)/float32(2) - (srcHeight*aspect)/float32(2))
	}

	destWidth := int(srcWidth * aspect)
	destHeight := int(srcHeight * aspect)

	//fmt.Printf("dst x:%d, y:%d w:%d, h:%d, perc:%f \n", destX, destY, destWidth, destHeight, aspect)

	tmpImage := image.NewRGBA(image.Rect(0, 0, destWidth, destHeight))
	draw.BiLinear.Scale(tmpImage, tmpImage.Bounds(), src, src.Bounds(), draw.Src, nil)

	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	//fill background
	draw.Draw(dst, dst.Bounds(), image.Black, image.ZP, draw.Src)

	//copy to result image
	draw.Copy(dst, image.Point{destX, destY}, tmpImage, tmpImage.Bounds(), draw.Src, nil)

	return dst
}

func drawWatermark(base image.Image, watermark image.Image) image.Image {

	watermarkWidth := watermark.Bounds().Size().X
	watermarkHeight := watermark.Bounds().Size().Y
	baseWidht := base.Bounds().Size().X
	baseHeight := base.Bounds().Size().Y

	columns := baseWidht / watermarkWidth
	rows := baseHeight / watermarkHeight
	if columns%2 == 0 {
		columns += 3
	} else {
		columns += 2
	}

	if rows%2 == 0 {
		rows += 3
	} else {
		rows += 2
	}

	//generate tile
	offsetX := int(0)
	offsetY := int(0)
	tmpImageWidht := watermarkWidth * columns
	tmpImageHeight := watermarkHeight * rows
	tmpImage := image.NewRGBA(image.Rect(0, 0, tmpImageWidht, tmpImageHeight))
	draw.Draw(tmpImage, tmpImage.Bounds(), image.Transparent, image.ZP, draw.Src)

	for row := 0; row < rows; row++ {
		offsetX = 0
		for column := 0; column < columns; column++ {
			draw.Copy(tmpImage, image.Point{offsetX, offsetY}, watermark, watermark.Bounds(), draw.Src, nil)
			offsetX += watermarkWidth
		}
		offsetY += watermarkHeight
	}

	//calculate offset
	destX := baseWidht/2 - tmpImageWidht/2
	destY := baseHeight/2 - tmpImageHeight/2

	dst := image.NewRGBA(base.Bounds())
	draw.Copy(dst, image.ZP, base, base.Bounds(), draw.Src, nil)

	draw.Copy(dst, image.Point{destX, destY}, tmpImage, tmpImage.Bounds(), draw.Over, nil)

	return dst
}

func decodeImage(base string) image.Image {
	baseFile, err := os.Open(base)
	if err != nil {
		fmt.Printf("Failed to open image:%s err:%s\n", base, err)
		return nil
	}

	usePngDecoder := strings.Compare(strings.ToLower(path.Ext(base)), ".png") == 0

	var baseImage image.Image
	if usePngDecoder {
		baseImage, err = png.Decode(baseFile)
	} else {
		baseImage, err = jpeg.Decode(baseFile)
	}
	defer baseFile.Close()

	if err != nil {
		fmt.Printf("Failed to decode image:%s err:%s\n", base, err)
		return nil
	}

	return baseImage
}

func addWatermark(base string, watermark string, resWidth int, resHeight int, outFileName string) {

	baseImage := decodeImage(base)
	if baseImage == nil {
		return
	}

	watermarkImage := decodeImage(watermark)
	if watermarkImage == nil {
		return
	}

	resizedImage := resize(baseImage, resWidth, resHeight)
	resultImage := drawWatermark(resizedImage, watermarkImage)

	outImageFile, err := os.OpenFile(outFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Printf("Failed to create: %s\n", err)
		return
	}

	png.Encode(outImageFile, resultImage)
}

//upload request handler
func upldoadHandler(res http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method)
	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("%v\n", r.Header)

	images := make(map[string]string)

	//read multipart data
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}

		if part.FileName() != "" {
			//save input files
			dstFile := "./" + path.Base(part.FileName())
			dst, err := os.OpenFile(dstFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
			if err != nil {
				fmt.Println(err)
				return
			}

			defer dst.Close()
			io.Copy(dst, part)

			images[part.FormName()] = dstFile
		}
	}

	outFileName := "result.png"
	base := images["uploadfile"]
	watermark := images["watermark"]

	//TODO: parse resolution parameters from request
	resWidth := 1024
	resHeight := 768

	addWatermark(base, watermark, resWidth, resHeight, outFileName)

	//write response
	fh, err := os.Open(outFileName)
	if err != nil {
		fmt.Println("Error opening file")
		return
	}
	defer fh.Close()

	stat, err := fh.Stat()
	if err != nil {
		fmt.Println("Error get stat of result image")
		return
	}
	buffer := make([]byte, stat.Size())
	io.ReadFull(fh, buffer)
	res.Header().Set("Content-Length", fmt.Sprint(stat.Size()))
	res.Write(buffer)

	fmt.Printf("Complete. Saved to:%s\n", outFileName)
}

func main() {
	http.HandleFunc("/watermark", upldoadHandler)
	err := http.ListenAndServe(":3210", nil)
	if err != nil {
		log.Fatal("Listen serv: ", err)
	}
}
