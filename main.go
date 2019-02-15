package main

import (
	"fmt"
	"image"
	"log"
	"math"
	"os"

	"github.com/chai2010/webp"
)

func getFilesize(path string) (size int64, err error) {
	fi, err := os.Stat(path)
	if err != nil {
		return
	}
	size = fi.Size()
	return
}

func compare(original image.Image, quality float32) (index float64, raw []byte, err error) {
	raw, err = webp.EncodeRGB(original, quality)
	if err != nil {
		return
	}
	decoded, err := webp.DecodeRGBA(raw)
	if err != nil {
		return
	}
	index = ssim(original, convertToGray(decoded))
	return
}

func save(p string, data []byte) (err error) {
	f, err := os.Create(p)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	f.Write(data)
	return
}

func main() {
	var maxQ float32 = 95
	var minQ float32 = 40
	target := 0.999
	loops := 6

	original := convertToGray(readImage("original.jpg")) // or a webp
	originalSize, err := getFilesize("original.jpg")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Original Size = %.2fKB\n", float32(originalSize)/1024)

	var bestImg []byte
	var bestSize = originalSize
	var bestQ float32
	var bestIndex float64
	var newSize int64
	var index float64
	for attempt := 1; attempt <= loops; attempt++ {
		var q = minQ + (maxQ-minQ)/2
		if minQ == maxQ {
			// fmt.Println("Tried all qualities")
			break
		}
		idx, data, err := compare(original, q)
		index = idx
		if err != nil {
			panic("Error when comparing images")
		}
		newSize = int64(len(data))
		fmt.Printf("[%v] Quality = %v, SSIM = %.5f, Size = %.2fKB\n", attempt, int(q), index, float32(newSize)/1024)

		if newSize >= originalSize {
			if index < target {
				fmt.Println("Cannot compress further")
				break
			} else {
				maxQ = float32(math.Max(float64(q-1), float64(minQ)))
			}
		} else {
			if index < target {
				minQ = float32(math.Min(float64(q+1), float64(maxQ)))
			} else if index > target {
				maxQ = float32(math.Max(float64(q-1), float64(minQ)))
			} else {
				fmt.Println("Found perfect compression")
				attempt = loops
			}
		}
		if newSize < bestSize && index >= target {
			bestImg = data
			bestSize = newSize
			bestQ = q
			bestIndex = index
		}
	}

	if len(bestImg) > 0 {
		fmt.Printf("Final image:\nQuality = %v, SSIM = %.5f, Size = %.2fKB\n", int(bestQ), bestIndex, float32(bestSize)/1024)
		fmt.Printf("%.1f%% of original, saved %.2fKB", float32(bestSize)/float32(originalSize)*100, float32(originalSize-bestSize)/1024)
		save("output.webp", bestImg)
	} else {
		fmt.Println("No new image")
	}
}
