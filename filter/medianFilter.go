package main

import (
	"flag"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"sort"
)

// check handles a potential error.
// It stops execution of the program ("panics") if an error has happened.
func check(err error) {
	if err != nil {
		panic(err)
	}
}

// makeMatrix makes and returns a 2D slice with the given dimensions.
func makeMatrix(height, width int) [][]uint8 {
	matrix := make([][]uint8, height)
	for i := range matrix {
		matrix[i] = make([]uint8, width)
	}
	return matrix
}

// makeImmutableMatrix takes an existing 2D matrix and wraps it in a getter closure.
func makeImmutableMatrix(matrix [][]uint8) func(y, x int) uint8 {
	return func(y, x int) uint8 {
		return matrix[y][x]
	}
}

// medianFilter applies the filter between the given x and y bounds on the given closure.
// medianFilter returns the section where the filter was applied as a 2D slice.
func medianFilter(startY, endY, startX, endX int, data func(y, x int) uint8) [][]uint8 {
	height := endY - startY
	width := endX - startX
	radius := 2
	midPoint := (5*5 + 1) / 2

	filteredMatrix := makeMatrix(height, width)
	filterValues := make([]int, 5*5)

	for i := radius + startY; i < endY-radius; i++ {
		for j := radius + startX; j < endX-radius; j++ {
			count := 0
			for k := i - radius; k <= i+radius; k++ {
				for l := j - radius; l <= j+radius; l++ {
					filterValues[count] = int(data(k, l))
					count++
				}
			}
			sort.Ints(filterValues)
			filteredMatrix[i-startY][j-startX] = uint8(filterValues[midPoint])
		}
	}
	return filteredMatrix
}

// getPixelData transfers an image.Image to a standard 2D slice.
func getPixelData(img image.Image) [][]uint8 {
	bounds := img.Bounds()
	pixels := makeMatrix(bounds.Dy(), bounds.Dx())

	curr := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
			pixels[y][x] = uint8(lum / 256)
			curr++
		}
	}
	return pixels
}

// loadImage opens a file and returns the contents as an image.Image.
func loadImage(filepath string) image.Image {
	existingImageFile, err := os.Open(filepath)
	check(err)
	defer func(existingImageFile *os.File) {
		err := existingImageFile.Close()
		if err != nil {

		}
	}(existingImageFile)

	img, _, err := image.Decode(existingImageFile)
	check(err)

	return img
}

// flattenImage takes a 2D slice and flattens it into a single 1D slice.
func flattenImage(image [][]uint8) []uint8 {
	height := len(image)
	width := len(image[0])

	flattenedImage := make([]uint8, 0, height*width)
	for i := 0; i < height; i++ {
		flattenedImage = append(flattenedImage, image[i]...)
	}
	return flattenedImage
}

// filter reads in a png image, applies the filter and outputs the result as a png image.
// filter is the function called by the tests in medianfilter_test.go
func filter(filepathIn, filepathOut string, threads int) {
	image.RegisterFormat("png", "PNG", png.Decode, png.DecodeConfig)
	image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)

	img := loadImage(filepathIn)
	bounds := img.Bounds()
	height := bounds.Dy()
	width := bounds.Dx()

	immutableData := makeImmutableMatrix(getPixelData(img))
	var newPixelData [][]uint8

	if threads == 1 {
		newPixelData = medianFilter(0, height, 0, width, immutableData)
	} else {

		//channelOne := make(chan [][]uint8)
		//channelTwo := make(chan [][]uint8)
		//channelThree := make(chan [][]uint8)
		//channelFour := make(chan [][]uint8)
		//
		//go func(ch chan<- [][]uint8) {
		//	result := medianFilter(0, 128, 0, width, immutableData)
		//	channelOne <- result
		//}(channelOne)
		//dataOne := <-channelOne
		//newPixelData = append(newPixelData, dataOne...)
		//
		//go func(ch chan<- [][]uint8) {
		//	result := medianFilter(128, 256, 0, width, immutableData)
		//	channelTwo <- result
		//}(channelTwo)
		//dataTwo := <-channelTwo
		//newPixelData = append(newPixelData, dataTwo...)
		//
		//go func(ch chan<- [][]uint8) {
		//	result := medianFilter(256, 384, 0, width, immutableData)
		//	channelThree <- result
		//}(channelThree)
		//dataThree := <-channelThree
		//newPixelData = append(newPixelData, dataThree...)
		//
		//go func(ch chan<- [][]uint8) {
		//	result := medianFilter(384, 512, 0, width, immutableData)
		//	channelFour <- result
		//}(channelFour)
		//dataFour := <-channelFour
		//newPixelData = append(newPixelData, dataFour...)

		channels := make([]chan [][]uint8, threads)

		for i := 0; i < threads; i++ {
			channels[i] = make(chan [][]uint8, 1)
			go func(start, end int, ch chan<- [][]uint8) {
				result := medianFilter(start, end, 0, width, immutableData)
				ch <- result
			}(i*(512/len(channels)), (i+1)*(512/len(channels)), channels[i])
		}

		for j := 0; j < threads; j++ {
			data := <-channels[j]
			newPixelData = append(newPixelData, data...)
		}
	}

	imout := image.NewGray(image.Rect(0, 0, width, height))
	imout.Pix = flattenImage(newPixelData)
	ofp, _ := os.Create(filepathOut)
	defer func(ofp *os.File) {
		err := ofp.Close()
		if err != nil {

		}
	}(ofp)
	err := png.Encode(ofp, imout)
	check(err)
}

func worker(startY, endY, startX, endX int, data func(y, x int) uint8, out chan<- [][]uint8) {
	filtered := medianFilter(startY, endY, startX, endX, data)
	out <- filtered
}

// main reads in the filepath flags or sets them to default values and calls filter().
func main() {
	var filepathIn string
	var filepathOut string
	var threads int

	flag.StringVar(
		&filepathIn,
		"in",
		"ship.png",
		"Specify the input file.")

	flag.StringVar(
		&filepathOut,
		"out",
		"out.png",
		"Specify the output file.")

	flag.IntVar(
		&threads,
		"threads",
		1,
		"Specify the number of worker threads to use.")

	flag.Parse()
	filter(filepathIn, filepathOut, threads)
}
