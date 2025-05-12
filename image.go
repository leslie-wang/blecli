package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"golang.org/x/image/bmp"
)

const (
	WIDTH      = 800
	WIDTH_HALF = 400
	HEIGHT     = 480
)

var palette = []color.Color{
	color.RGBA{0, 0, 0, 255},       // Black
	color.RGBA{255, 255, 255, 255}, // White
	color.RGBA{0, 255, 0, 255},     // Green
	color.RGBA{0, 0, 255, 255},     // Blue
	color.RGBA{255, 0, 0, 255},     // Red
	color.RGBA{255, 255, 0, 255},   // Yellow
	color.RGBA{255, 128, 0, 255},   // Orange
}

func closestColor(c color.Color) color.Color {
	r1, g1, b1, _ := c.RGBA()
	r1 >>= 8
	g1 >>= 8
	b1 >>= 8
	minDist := uint32(1<<32 - 1)
	var closest color.Color
	for _, pc := range palette {
		r2, g2, b2, _ := pc.RGBA()
		r2 >>= 8
		g2 >>= 8
		b2 >>= 8
		// Euclidean distance (no sqrt)
		dr := int32(r1 - r2)
		dg := int32(g1 - g2)
		db := int32(b1 - b2)
		dist := uint32(dr*dr + dg*dg + db*db)
		if dist < minDist {
			minDist = dist
			closest = pc
		}
	}
	return closest
}

func clamp(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func mapColorByRGB(r, g, b uint32) (color byte) {
	if r == 0 && g == 0 && b == 0 {
		// Image[x+(y* bmpInfoHeader.biWidth )] =  0;//Black
		color = 0
	} else if r == 255 && g == 255 && b == 255 {
		// Image[x+(y* bmpInfoHeader.biWidth )] =  1;//White
		color = 1
	} else if r == 0 && g == 255 && b == 0 {
		// Image[x+(y* bmpInfoHeader.biWidth )] =  2;//Green
		color = 2
	} else if r == 0 && g == 0 && b == 255 {
		// Image[x+(y* bmpInfoHeader.biWidth )] =  3;//Blue
		color = 3
	} else if r == 255 && g == 0 && b == 0 {
		// Image[x+(y* bmpInfoHeader.biWidth )] =  4;//Red
		color = 4
	} else if r == 255 && g == 255 && b == 0 {
		// Image[x+(y* bmpInfoHeader.biWidth )] =  5;//Yellow
		color = 5
	} else if r == 255 && g == 128 && b == 0 {
		// Image[x+(y* bmpInfoHeader.biWidth )] =  6;//Orange
		color = 6
	}
	return
}

/*
func mapColorByIndex(color byte) (r, g, b uint8) {
	switch color {
	case 0: //Black
		return 0, 0, 0
	case 1: // White
		return 255, 255, 255
	case 2: // Green
		return 0, 255, 0
	case 3: // Blue
		return 255, 0, 0
	case 4: // Red
		return 0, 0, 255
	case 5: // Yellow
		return 0, 255, 255
	case 6: // Orange
		return 0, 128, 255
	default: // Black
		return 0, 0, 0
	}
}
*/

func floydSteinbergDither(img image.Image) (*image.Paletted, []byte) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	rgba := image.NewNRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	rawData := make([]byte, width*height/2)
	dithered := image.NewPaletted(bounds, palette)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*rgba.Stride + x*4
			oldR := int(rgba.Pix[i])
			oldG := int(rgba.Pix[i+1])
			oldB := int(rgba.Pix[i+2])
			oldColor := color.RGBA{uint8(oldR), uint8(oldG), uint8(oldB), 255}
			newColor := closestColor(oldColor)
			nr, ng, nb, _ := newColor.RGBA()
			nr >>= 8
			ng >>= 8
			nb >>= 8
			rgba.Set(x, y, newColor)
			dithered.Set(x, y, newColor)

			idx := x/2 + y*WIDTH_HALF
			c := mapColorByRGB(nr, ng, nb)

			data := rawData[idx]
			data = data & (^(0xF0 >> ((x % 2) * 4))) //Clear first, then set value
			rawData[idx] = data | ((c << 4) >> ((x % 2) * 4))

			errR := oldR - int(nr)
			errG := oldG - int(ng)
			errB := oldB - int(nb)

			// Diffuse the error
			for _, offset := range [][3]int{
				{1, 0, 7},
				{-1, 1, 3},
				{0, 1, 5},
				{1, 1, 1},
			} {
				dx, dy, factor := offset[0], offset[1], offset[2]
				nx, ny := x+dx, y+dy
				if nx >= 0 && nx < width && ny >= 0 && ny < height {
					ni := ny*rgba.Stride + nx*4
					rgba.Pix[ni+0] = clamp(int(rgba.Pix[ni+0]) + errR*factor/16)
					rgba.Pix[ni+1] = clamp(int(rgba.Pix[ni+1]) + errG*factor/16)
					rgba.Pix[ni+2] = clamp(int(rgba.Pix[ni+2]) + errB*factor/16)
				}
			}
		}
	}
	return dithered, rawData
}

func resize(src image.Image) image.Image {
	dstW, dstH := WIDTH, HEIGHT
	if src.Bounds().Max.Y-src.Bounds().Min.Y > src.Bounds().Max.X-src.Bounds().Min.X {
		dstW, dstH = HEIGHT, WIDTH
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			srcX := srcBounds.Min.X + x*srcW/dstW
			srcY := srcBounds.Min.Y + y*srcH/dstH
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

func convertImage(c *cli.Context) error {
	if len(c.Args()) != 1 {
		return errors.New("Usage: convert input.jpg")
	}

	inputFilename := c.Args()[0]
	outputFilename := inputFilename + ".bmp"
	epaperFilename := inputFilename + ".epa"

	inputFile, err := os.Open(inputFilename)
	if err != nil {
		return nil
	}
	defer inputFile.Close()

	srcImg, _, err := image.Decode(inputFile)
	if err != nil {
		return nil
	}

	resized := resize(srcImg)
	result, epaperResult := floydSteinbergDither(resized)

	outputFile, err := os.Create(outputFilename)
	if err != nil {
		return nil
	}
	defer outputFile.Close()

	err = bmp.Encode(outputFile, result)
	if err != nil {
		return err
	}

	fmt.Printf("Dithering complete. Output saved to %s and %s\n",
		outputFilename, epaperFilename)

	os.WriteFile(epaperFilename, epaperResult, 0644)

	return nil
}

func convertRaw(c *cli.Context) error {
	var (
		data []byte
		err  error
	)
	outputFilename := "testdata.bmp"

	if len(c.Args()) == 0 {
		data = testData
	} else {
		outputFilename = c.Args()[0] + ".bmp"

		data, err = os.ReadFile(c.Args()[0])
		if err != nil {
			return err
		}
	}

	width, height := WIDTH, HEIGHT
	bounds := image.Rect(0, 0, width, height)
	img := image.NewPaletted(bounds, palette)

	pixelIndex := 0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x += 2 {
			byteVal := data[pixelIndex]
			hi := (byteVal & 0xF0) >> 4
			lo := byteVal & 0x0F

			img.Set(x, y, palette[hi%8])
			img.Set(x+1, y, palette[lo%8])
			pixelIndex++
		}
	}

	outputFile, err := os.Create(outputFilename)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	err = bmp.Encode(outputFile, img)
	if err != nil {
		return err
	}

	println("Convertion complete. Output saved to", outputFilename)
	return nil
}
