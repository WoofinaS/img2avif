package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/b4b4r07/go-pipe"
	flags "github.com/spf13/pflag"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
)

var workers int
var del bool
var pixfmt, advArgs string
var aomargs = []string{
	"--allintra", "--cpu-used=3", "--end-usage=q", "--cq-level=18", "--deltaq-mode=3", "-", "--ivf", "-o", "-",
}

func init() {
	flags.CommandLine.SortFlags = false
	flags.StringVar(&advArgs, "args", "", "Specify any arguments arguments to pass to aomenc")
	flags.StringVar(&pixfmt, "pix-format", "yuv444p10le", "Sets the pixel format")
	flags.IntVar(&workers, "workers", runtime.NumCPU()/2, "Specify the number of workers to use")
	flags.BoolVar(&del, "delete", false, "delete files after transcoding")
	flags.Parse()
	if len(flags.Args()) == 0 {
		flags.Usage()
		os.Exit(0)
	}
	if workers <= 0 {
		fmt.Println("Requires at least 1 worker")
		os.Exit(1)
	}
	if workers > len(flag.Args()) {
		workers = len(flags.Args())
	}
	aomargs = append(aomargs, fmt.Sprintf("--threads=%d", int(float32(runtime.NumCPU()/workers)*1.5)))
	aomargs = append(aomargs, strings.Fields(advArgs)...)
}

func main() {
	wg := new(sync.WaitGroup)
	wg.Add(workers)
	files := make(chan string)
	for i := 1; i <= workers; i++ {
		go convert(files, wg)
	}
	for _, file := range flags.Args() {
		files <- file
	}
	close(files)
	wg.Wait()
}

func convert(files chan string, wg *sync.WaitGroup) {
	for file := range files {
		output := file[0:len(file)-len(path.Ext(file))] + ".avif"
		avifenc(file, output)
	}
	wg.Done()
}

func avifenc(input, output string) {
	var b bytes.Buffer
	err := pipe.Command(&b,
		exec.Command("ffmpeg", "-i", input, "-strict", "-2", "-pix_fmt", pixfmt, "-f", "yuv4mpegpipe", "-"),
		exec.Command("aomenc", aomargs...),
		exec.Command("MP4Box", "-add-image", "-:primary", "-ab", "avif", "-ab", "miaf", "-new", output),
	)
	if err != nil {
		fmt.Printf("Failed to convert \"%s\"\n", input)
		fmt.Println(err)
	} else {
		if del {
			os.Remove(input)
		}
	}
}
