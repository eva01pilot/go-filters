package main

import (
	"fmt"
	"go-filters/filters"
	"go-filters/fonts"
	"go-filters/video"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

type FrameRouter struct {
	reader  io.WriteCloser
	channel chan struct {
		img   image.Image
		index int
	}
	filters         []filters.Filter
	dynamic_filters []filters.DynamicFilter
}

func (r *FrameRouter) Process(img image.Image, current_frame int) {
	rgba := img.(*image.RGBA)
	for _, selected_filter := range r.dynamic_filters {
		selected_filter.Filter(rgba, current_frame)

	}

	r.channel <- struct {
		img   image.Image
		index int
	}{img: rgba, index: current_frame}

}

var rootCmd = &cobra.Command{
	Use:   "go-filters [text]",
	Short: "CLI that applies filters on mp4 video frames and outputs new one",
	Args:  cobra.ArbitraryArgs,
	Run:   runCliApp,
}

var input *string
var output *string

func init() {
	input = rootCmd.Flags().String("input", "", "Path to input mp4 video file")
	output = rootCmd.Flags().String("output", "", "Path to output mp4 video file")
}

func runCliApp(command *cobra.Command, args []string) {

	_, err := os.Stat(*output)
	if err == nil {
		os.Remove(*output)
	}
	cmd := exec.Command("ffmpeg",
		"-f", "image2pipe",
		"-vcodec", "png",
		"-r", "30",
		"-i", "-",
		"-c:v", "libx264",
		"-pix_fmt", "yuv444p",
		"-crf", "18",
		"-preset", "slow",
		"-x264-params", "aq-mode=3:aq-strength=1.2:chroma_qp_offset=-2",
		*output)
	pipe, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	writech := make(chan struct {
		img   image.Image
		index int
	}, 8)

	router := FrameRouter{reader: pipe, channel: writech}

	grayScale := filters.GrayscaleFilter{}
	gaussianBlur := filters.GaussianBlur{}
	asciiFilter := filters.AsciiFilter{}

	router.dynamic_filters = append(router.dynamic_filters, &grayScale, &gaussianBlur, &asciiFilter)

	numFrames, err := video.GetFrameCount(*input)
	if err != nil {
		panic(err)
	}

	done := make(chan struct{}) // To signal when writing is complete

	go func() {
		defer close(done)
		expected := 0
		doneMap := make(map[int]image.Image, numFrames)

		for frame := range writech {
			idx := frame.index
			doneMap[idx] = frame.img

			for {
				img, ok := doneMap[expected]
				if !ok {
					break
				}

				err = png.Encode(pipe, img)
				if err != nil {
					panic(err)
				}

				delete(doneMap, expected)
				expected++
			}

		}
	}()

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	err = video.Decode(*input, &router)
	if err != nil {
		panic(err)
	}

	close(writech)

	// Wait for writer goroutine to finish all encoding
	<-done

	// Close ffmpeg stdin to signal end of stream
	pipe.Close()

	cmd.Wait()

}
func main() {

	fonts.CreateASCIISprites(8)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
