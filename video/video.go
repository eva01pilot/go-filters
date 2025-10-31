package video

import (
	"bufio"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"runtime"

	//"runtime"
	"strconv"
	"strings"
	"sync"
)

type FrameProcessor interface {
	Process(frame image.Image, current_frame int)
}

type ImageJob chan struct {
	img   image.Image
	index int
}

func GetFrameCount(video_path string) (int, error) {
	frame_count_bytes, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-count_packets", "-show_entries", "stream=nb_read_packets",
		"-of", "csv=p=0", video_path).Output()

	if err != nil {
		return 0, err
	}

	stringified := string(frame_count_bytes)

	frame_count, err := strconv.Atoi(strings.TrimSpace(stringified))

	if err != nil {
		return 0, err
	}

	return frame_count, nil

}

func Decode(video_path string, processor FrameProcessor) error {

	frame_count, err := GetFrameCount(video_path)
	fmt.Println(frame_count)

	cmd := exec.Command(
		"ffmpeg",
		"-hwaccel", "auto",
		"-ss", "10",
		"-t", "10",
		"-i", video_path,
		"-f", "image2pipe",
		"-vcodec", "png",
		"-",
	)
	cmd.Stderr = os.Stderr
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	err = cmd.Start()
	if err != nil {
		return err
	}
	defer pipe.Close()

	r := bufio.NewReader(pipe)

	var wg sync.WaitGroup
	var currentFrame int

	jobs := make(ImageJob, 8)

	concurrency := runtime.NumCPU()

	for range concurrency {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for job := range jobs {
				processor.Process(job.img, job.index)
			}
		}()
	}

	for {
		img, err := png.Decode(r)

		if err != nil {
			if err == io.EOF {
				break
			}

			if err == io.ErrUnexpectedEOF {
				break
			}
			panic(err)
		}
		jobs <- struct {
			img   image.Image
			index int
		}{img: img, index: currentFrame}

		currentFrame++
	}

	close(jobs)

	wg.Wait()
	return nil
}
