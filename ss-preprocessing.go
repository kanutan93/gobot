package main

import (
	"github.com/vova616/screenshot"
	"github.com/lazywei/go-opencv/opencv"
	"image"
	"os"
	"image/png"
	"time"
	"fmt"
	"github.com/otiai10/gosseract"
	"strings"
	"io/ioutil"
	"encoding/json"
)

const (
	hpImageFilename = "hpImage.png"
	mpImageFilename = "mpImage.png"
	targetsImageFilename = "targets.png"
	colorConf = "color.json"
)

type BoxesCoords struct {
	HpBox BoxCoords
	MpBox BoxCoords
}

type BoxCoords struct {
	X      int
	Y      int
	Width  int
	Height int
}

type Color struct {
	Lower []float64
	Upper []float64
	ColorIndent float64
}

func readCoordsConf(filename string) BoxesCoords {
	coordsConfig, err := ioutil.ReadFile(filename)
	check(err)
	var coords BoxesCoords
	json.Unmarshal([]byte(coordsConfig), &coords)
	return coords
}

func readColorConf(filename string) Color {
	colorConfig, err := ioutil.ReadFile(filename)
	check(err)
	var color Color
	json.Unmarshal([]byte(colorConfig), &color)
	return color
}

func createScreenshot(filename string) {
	img, err := screenshot.CaptureScreen()
	check(err)
	myImg := image.Image(img)
	file, err := os.Create(filename)
	check(err)
	png.Encode(file, myImg)
	fmt.Println("Screenshot created at: " + time.Now().String())
}

func getPoints(filename string, coords *BoxesCoords) map[string]string {
	img := opencv.LoadImage(filename)
	defer img.Release()
	hpImage := getBoxFilename(hpImageFilename, img, &coords.HpBox)
	mpImage := getBoxFilename(mpImageFilename, img, &coords.MpBox)
	points := map[string]string{"hp": hpImage, "mp": mpImage}
	return points
}

func getBoxFilename(pointsImageFilename string, image *opencv.IplImage, boxCoords *BoxCoords) string {
	box := opencv.Crop(image, boxCoords.X, boxCoords.Y, boxCoords.Width, boxCoords.Height)
	opencv.SaveImage(pointsImageFilename, box, nil)
	box.Release()
	return pointsImageFilename
}

func recognizePoints(points map[string]string) map[string]string {
	for key, value := range points {
		points[key] = strings.TrimSpace(gosseract.Must(gosseract.Params{Src: value, Languages: "eng"}))
	}
	return points
}

func drawTargetBox(img *opencv.IplImage, x int, y int) {
	opencv.Rectangle(img,
		opencv.Point{x - 75, y + 50},
		opencv.Point{x + 50, y + 150},
		opencv.NewScalar(170, 255, 102, 255),
		3, 4, 0)
}

func findTarget(filename string, color *Color) map[string]int{
	img := opencv.LoadImage(filename)
	defer img.Release()

	coords := map[string]int{"x": 0, "y": 0} //83 127 65
	colorUpper := [4]float64{color.Upper[0], color.Upper[1], color.Upper[2], 0} //верхняя граница зеленого цвета BGR
	colorLower := [4]float64{color.Lower[0], color.Lower[1], color.Lower[2], 0} //нижняя граница зеленого цвета BGR

	for x := 0; x < img.Width(); x++ {
		for y := 0; y < img.Height(); y++ {
			scalar := img.Get2D(x, y)
			scalarArray := scalar.Val() //Массив пикселей изображения
			if((scalarArray[0] < scalarArray[1] - 70) && (scalarArray[1] > scalarArray[2] - 70)){ // Если зеленого больше
				if (scalarArray[0] >= colorLower[0]) && (scalarArray[0] <= colorUpper[0]) { //, то ищем пиксель в заданном выше диапозоне
					if (scalarArray[1] >= colorLower[1]) && (scalarArray[1] <= colorUpper[1]) {
						if (scalarArray[2] >= colorLower[2]) && (scalarArray[2] <= colorUpper[2]) {
							coords["x"] = x
							coords["y"] = y
							break
						}
					}
				}
			}
		}
	}
	drawTargetBox(img, coords["x"], coords["y"])
	opencv.SaveImage(targetsImageFilename, img, nil)
	return coords
}
