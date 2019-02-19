package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"log"
	"math"
	"net/http"
	"os"

	"github.com/chai2010/webp"
)

func isWebp(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	// Get the content
	buffer := make([]byte, 512)
	_, err = f.Read(buffer)
	if err != nil {
		return false
	}
	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)
	return contentType == "image/webp"
}

func getFilesize(path string) (size int64, err error) {
	fi, err := os.Stat(path)
	if err != nil {
		return
	}
	size = fi.Size()
	return
}

func compare(original image.Image, quality int) (index float64, raw []byte, err error) {
	raw, err = webp.EncodeRGB(original, float32(quality))
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

func copyFile(src string, dest string) (nBytes int64, err error) {
	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err = io.Copy(destination, source)
	return nBytes, err
}

func checkArgs(src string, dest string, force bool, max int, min int, target float64, loops int) bool {
	var msg string
	if _, err := os.Stat(src); os.IsNotExist(err) {
		msg = "Source image '" + src + "' does not exists."
	}
	if !force {
		if _, err := os.Stat(dest); os.IsExist(err) {
			msg = "Destiation path '" + dest + "' already exists. Use -f to overwrite."
		}
	}
	if dest == "" {
		msg = "Please specify a destination path"
	}
	if max < 1 || max > 100 {
		msg = "Maximum quality has to be between 1 and 100."
	}
	if min < 0 || min > 99 {
		msg = "Minimum quality has to be between 0 and 99."
	}
	if target <= 0 || target > 1 {
		msg = "Target has to be between 0 and 99."
	}
	if loops <= 0 {
		msg = "Loops has to be more than 0"
	}
	if msg == "" {
		return true
	}
	fmt.Println("* Error: " + msg)
	return false
}

func main() {
	var maxQ int
	var minQ int
	var target float64
	var loops int
	var force bool
	var help bool
	flag.IntVar(&maxQ, "max", 95, "Maximum quality")
	flag.IntVar(&minQ, "min", 40, "Minimum quality")
	flag.Float64Var(&target, "t", 0.999, "Target minimum SSIM")
	flag.IntVar(&loops, "l", 6, "Numer of tries")
	flag.BoolVar(&force, "f", false, "Whether to overwrite the output image if it already exists")
	flag.BoolVar(&help, "h", false, "Print this help message")
	flag.Parse()
	src := flag.Arg(0)
	dest := flag.Arg(1)

	// check args
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ./webpre src dest [options]\n")
		flag.PrintDefaults()
	}
	if help {
		flag.Usage()
		return
	}
	if !checkArgs(src, dest, force, maxQ, minQ, target, loops) {
		flag.Usage()
		os.Exit(1)
	}

	original := readImage(src) // or a webp
	originalSize, err := getFilesize(src)
	originalGray := convertToGray(original)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Original Size = %.2fKB\n", float32(originalSize)/1024)

	var bestSize = originalSize
	var bestQ int
	var bestIndex float64
	var fallbackQ int
	var fallbackSize int64
	var fallbackIndex float64
	for attempt := 1; attempt <= loops; attempt++ {
		var q = minQ + (maxQ-minQ)/2
		if minQ == maxQ {
			// fmt.Println("Tried all qualities")
			break
		}
		index, data, err := compare(originalGray, q)
		if err != nil {
			panic("Error when comparing images")
		}
		newSize := int64(len(data))
		fmt.Printf("[%v] Quality = %v, SSIM = %.5f, Size = %.2fKB\n", attempt, q, index, float32(newSize)/1024)

		if newSize >= originalSize {
			if index < target {
				fmt.Println("* Cannot achieve target SSIM by compressing further")
				attempt = loops
			} else {
				maxQ = int(math.Max(float64(q-1), float64(minQ)))
			}
		} else {
			if index < target {
				minQ = int(math.Min(float64(q+1), float64(maxQ)))
			} else if index > target {
				maxQ = int(math.Max(float64(q-1), float64(minQ)))
			} else {
				fmt.Println("Found perfect compression")
				attempt = loops
			}
		}
		if newSize < bestSize && index >= target {
			bestSize = newSize
			bestQ = q
			bestIndex = index
		}
		// fallback
		if fallbackSize == 0 {
			fallbackSize = newSize
		}
		if newSize <= originalSize && newSize > fallbackSize {
			// when it's smaller than original we find the biggest
			fallbackSize = newSize
			fallbackQ = q
			fallbackIndex = index
		} else if newSize > originalSize && newSize < fallbackSize {
			// when it's bigger than ooriginal we find the smallest
			fallbackSize = newSize
			fallbackQ = q
			fallbackIndex = index
		}
	}

	if bestSize < originalSize {
		data, err := webp.EncodeRGB(original, float32(bestQ))
		if err != nil {
			panic(err)
		}
		save(dest, data)
		fmt.Printf("Final image:\nQuality = %v, SSIM = %.5f, Size = %.2fKB\n", bestQ, bestIndex, float32(bestSize)/1024)
		fmt.Printf("%.1f%% of original, saved %.2fKB", float32(bestSize)/float32(originalSize)*100, float32(originalSize-bestSize)/1024)
	} else {
		if isWebp(src) {
			fmt.Println("* Can't find target SSIM, copying oringal image")
			_, err := copyFile(src, dest)
			if err != nil {
				panic(err)
			}
		} else {
			fmt.Println("* Can't find target SSIM, falling back to closest match")
			fmt.Printf("Final image:\nQuality = %v, SSIM = %.5f, Size = %.2fKB\n", fallbackQ, fallbackIndex, float32(fallbackSize)/1024)
			fmt.Printf("%.1f%% of original, saved %.2fKB", float32(fallbackSize)/float32(originalSize)*100, float32(originalSize-fallbackSize)/1024)
			data, err := webp.EncodeRGB(original, float32(fallbackQ))
			if err != nil {
				panic(err)
			}
			save(dest, data)
		}
	}
}
