package main
import (
  "fmt"
  "os"
  "log"
  "image"
  "image/color"
  "image/jpeg"
  "math"
  "errors"
)

// Default SSIM constants
var (
  L = 255.0
  K1 = 0.01
  K2 = 0.03
  C1 = math.Pow((K1*L), 2.0)
  C2 = math.Pow((K2*L), 2.0)
)

func handleError(err error) {
  if err != nil {
    log.Fatal(err)
  }
}

// Given a path to an image file, read and return as
// an image.Image
func readImage(fname string) image.Image {
  file, err := os.Open(fname)
  handleError(err)
  defer file.Close()

  img, _, err := image.Decode(file)
  handleError(err)
  return img
}

// Convert an Image to grayscale which
// equalize RGB values
func convertToGray(originalImg image.Image) image.Image {
  bounds := originalImg.Bounds()
  w, h := dim(originalImg)

  grayImg := image.NewGray(bounds)

  for x := 0; x < w; x++ {
    for y := 0; y < h; y++ {
      originalColor := originalImg.At(x, y)
      grayColor := color.GrayModel.Convert(originalColor)
      grayImg.Set(x, y, grayColor)
    }
  }

  return grayImg
}

// Write an image.Image to a jpg file of quality 100
func writeImage(img image.Image, path string) {
  w, err := os.Create(path+".jpg")
  handleError(err)
  defer w.Close()

  quality := jpeg.Options{100}
  jpeg.Encode(w, img, &quality)
}

// Convert uint32 R value to a float. The returnng
// float will have a range of 0-255
func getPixVal(c color.Color) float64 {
  r, _, _, _ := c.RGBA()
  return float64(r >> 8)
}

// Helper function that return the dimension of an image
func dim(img image.Image) (w, h int) {
  w, h = img.Bounds().Max.X, img.Bounds().Max.Y
  return
}

// Check if two images have the same dimension
func equalDim(img1, img2 image.Image) bool {
  w1, h1 := dim(img1)
  w2, h2 := dim(img2)
  return (w1 == w2) && (h1 == h2)
}

// Given an Image, calculate the mean of its 
// pixel values
func mean(img image.Image) float64 {
  w, h := dim(img)
  n := float64((w*h)-1)
  sum := 0.0

  for x := 0; x < w; x++ {
    for y := 0; y < h; y++ {
      sum += getPixVal(img.At(x, y))
    }
  }
  return sum/n
}

// Compute the standard deviation with pixel values of Image
func stdev(img image.Image) float64  {
  w, h := dim(img)

  n := float64((w*h)-1)
  sum := 0.0
  avg := mean(img)

  for x := 0; x < w; x++ {
    for y := 0; y < h; y++ {
      pix := getPixVal(img.At(x, y))
      sum += math.Pow((pix - avg), 2.0)
    }
  }
  return math.Sqrt(sum/n)
}

// Calculate the covariance of 2 images
func covar(img1, img2 image.Image) (c float64,  err error) {
  if !equalDim(img1, img2) {
    err = errors.New("Images must have same dimension")
    return
  }
  avg1 := mean(img1)
  avg2 := mean(img2)
  w, h := dim(img1)
  sum := 0.0
  n := float64((w*h)-1)

  for x := 0; x < w; x++ {
    for y := 0; y < h; y++ {
      pix1 := getPixVal(img1.At(x, y))
      pix2 := getPixVal(img2.At(x, y))
      sum += (pix1 - avg1)*(pix2 - avg2)
    }
  }
  c = sum/n
  return
}

func ssim(x, y image.Image) float64 {
  avg_x := mean(x)
  avg_y := mean(y)

  stdev_x := stdev(x)
  stdev_y := stdev(y)

  cov, err := covar(x, y)
  handleError(err)

  numerator := ((2.0 * avg_x * avg_y) + C1) * ((2.0 * cov) + C2)
  denominator := ( math.Pow(avg_x, 2.0) + math.Pow(avg_y, 2.0) + C1) *
                 ( math.Pow(stdev_x, 2.0) + math.Pow(stdev_y, 2.0) + C2 )

  return numerator/denominator
}

func main() {
  img := convertToGray(readImage("original.jpg"))
  img2 := convertToGray(readImage("fake.jpg"))

  c, err := covar(img, img2)
  handleError(err)
  
  index := ssim(img, img2)

  // fmt.Printf("AVG   %f\n", mean(img))
  // fmt.Printf("STDEV %f\n", stdev(img))
  // fmt.Printf("COV   %f\n", c)
  _ = c
  fmt.Printf("\nSSIM = %f\n", index)
}

