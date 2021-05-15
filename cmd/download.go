/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/cheggaaa/pb"
	"github.com/spf13/cobra"
)

var Url string

func init() {
	rootCmd.AddCommand(downloadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	downloadCmd.PersistentFlags().String("foo", "", "A help for foo")
	downloadCmd.PersistentFlags().StringVarP(&Url, "url", "u", "", "Download URL")
	downloadCmd.PersistentFlags().BoolP("remove-existing", "r", false, "Remove file if exists")
	downloadCmd.PersistentFlags().Int64P("parts-count", "c", 4, "Use http-ranges (if supported) and divide to multiple of parts.")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// downloadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a file efficiently",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		partsCount, err := cmd.Flags().GetInt64("parts-count")
		if err != nil {
			log.Fatal(err)
		}
		if Url == "" {
			// downloadUrl = "https://github.com/helm/helm/archive/refs/tags/v3.5.4.zip"
			Url = "https://releases.hashicorp.com/terraform/0.15.3/terraform_0.15.3_linux_amd64.zip"
		}
		removeExisting, err := cmd.Flags().GetBool("remove-existing")
		if err != nil {
			log.Fatal(err)
		}

		downloadFunc(Url, partsCount, removeExisting)
		// Based on: https://github.com/jacklin293/golang-parallel-download-with-accept-ranges
	},
}

type Worker struct {
	Url       string
	File      *os.File
	Count     int64
	SyncWG    sync.WaitGroup
	TotalSize int64
	Progress
}

type Progress struct {
	Pool *pb.Pool
	Bars []*pb.ProgressBar
}

func downloadFunc(downloadUrl string, partsCount int64, removeExisting bool) {
	var t = flag.Bool("t", false, "file name with datetime")
	flag.Parse()

	// Get header from the url
	log.Println("Url:", downloadUrl)
	file_size, ranges_supported, err := getSizeAndCheckRangeSupport(downloadUrl)
	handleError(err)

	if file_size >= 100000 {
		log.Printf("File size: %sBytes\n", bytefmt.ByteSize(uint64(file_size)))
	} else {
		log.Printf("File size: %sytes\n", bytefmt.ByteSize(uint64(file_size)))
	}

	var file_path string
	if *t {
		file_path = filepath.Dir(os.Args[0]) + string(filepath.Separator) + strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + getFileName(downloadUrl)
	} else {
		file_path = filepath.Dir(os.Args[0]) + string(filepath.Separator) + getFileName(downloadUrl)
	}
	log.Printf("Local path: %s\n", file_path)

	if _, err := os.Stat(file_path); err == nil {
		if removeExisting {
			log.Printf("File exists, removing ...")
			os.Remove(file_path)
			log.Printf("Successfully removed existing file")
		} else {
			log.Println("File already exists, terminating")
			os.Exit(0)
		}
	}

	if err != nil {
		handleError(err)
	}
	f, err := os.OpenFile(file_path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		f.Close()
		handleError(err)
	}

	defer f.Close()

	if !ranges_supported {
		log.Println("Target does NOT support http-ranges, using a single worker")
		partsCount = int64(1)
	}

	// New worker struct to download file
	var worker = Worker{
		Url:       downloadUrl,
		File:      f,
		Count:     partsCount,
		TotalSize: file_size,
	}

	var start, end int64
	var partial_size = int64(file_size / partsCount)
	now := time.Now().UTC()
	for num := int64(0); num < worker.Count; num++ {
		// New sub progress bar (give it 0 at first for new instance and assign real size later on.)
		bar := pb.New(0).Prefix(fmt.Sprintf("Part %d  0%% ", num))
		bar.ShowSpeed = true
		bar.SetMaxWidth(100)
		bar.SetUnits(pb.U_BYTES_DEC)
		bar.SetRefreshRate(time.Second)
		bar.ShowPercent = true
		worker.Progress.Bars = append(worker.Progress.Bars, bar)

		if num == worker.Count-1 {
			end = file_size // last part
		} else {
			end = start + partial_size
		}
		// log.Println(num, start, end) // debugging

		worker.SyncWG.Add(1)
		go worker.writeRange(num, start, end-1)
		start = end
	}
	worker.Progress.Pool, err = pb.StartPool(worker.Progress.Bars...)
	if err != nil {
		worker.File.Close()
		handleError(err)
	}
	worker.SyncWG.Wait()
	worker.Progress.Pool.Stop()
	worker.File.Close() // final close
	log.Println("Elapsed time:", time.Since(now))
	log.Println("Done!")
	blockForWindows()
}

func (w *Worker) writeRange(part_num int64, start int64, end int64) {
	var written int64
	body, size, err := w.getRangeBody(start, end)
	if err != nil {
		w.File.Close()
		log.Fatalf("Part %d request error: %s\n", part_num, err.Error())
	}
	defer body.Close()
	defer w.Bars[part_num].Finish()
	defer w.SyncWG.Done()

	// Assign total size to progress bar
	w.Bars[part_num].Total = size

	// New percentage flag
	percent_flag := map[int64]bool{}

	// make a buffer to keep chunks that are read
	buf := make([]byte, 32*1024)
	for {
		nr, er := body.Read(buf)
		if nr > 0 {
			nw, err := w.File.WriteAt(buf[0:nr], start)
			if err != nil {
				log.Fatalf("Part %d occured error: %s.\n", part_num, err.Error())
			}
			if nr != nw {
				log.Fatalf("Part %d occured error of short writiing.\n", part_num)
			}

			start = int64(nw) + start
			if nw > 0 {
				written += int64(nw)
			}

			// Update written bytes on progress bar
			w.Bars[int(part_num)].Set64(written)

			// Update current percentage on progress bars
			p := int64(float32(written) / float32(size) * 100)
			_, flagged := percent_flag[p]
			if !flagged {
				percent_flag[p] = true
				w.Bars[int(part_num)].Prefix(fmt.Sprintf("Part %d  %d%% ", part_num, p))
			}
		}
		if er != nil {
			if er.Error() == "EOF" {
				if size == written {
					// Download successfully
				} else {
					handleError(errors.New(fmt.Sprintf("Part %d unfinished.\n", part_num)))
				}
				break
			}
			handleError(errors.New(fmt.Sprintf("Part %d occured error: %s\n", part_num, er.Error())))
		}
	}
}

func (w *Worker) getRangeBody(start int64, end int64) (io.ReadCloser, int64, error) {
	var client http.Client
	req, err := http.NewRequest("GET", w.Url, nil)
	// req.Header.Set("cookie", "")
	// log.Printf("Request header: %s\n", req.Header)
	if err != nil {
		return nil, 0, err
	}

	// Set range header
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	size, err := strconv.ParseInt(resp.Header["Content-Length"][0], 10, 64)
	return resp.Body, size, err
}

func getSizeAndCheckRangeSupport(url string) (size int64, ranges_supported bool, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	// req.Header.Set("cookie", "")
	// log.Printf("Request header: %s\n", req.Header)
	res, err := client.Do(req)
	if err != nil {
		return
	}
	// log.Printf("Response header: %v\n", res.Header)
	header := res.Header
	if len(header["Content-Length"]) == 0 {
		log.Fatalln("Server does not support Content-Length")
	}
	accept_ranges, supported := header["Accept-Ranges"]
	if !supported {
		size, err = strconv.ParseInt(header["Content-Length"][0], 10, 64)
		if err != nil {
			log.Fatal(err.Error())
		}
		return size, false, nil // , errors.New("Doesn't support header `Accept-Ranges`.")
	} else if supported && accept_ranges[0] != "bytes" {
		return 0, false, errors.New("support `Accept-Ranges`, but value is not `bytes`")
	}
	size, err = strconv.ParseInt(header["Content-Length"][0], 10, 64)
	return size, true, err
}

func getFileName(downloadUrl string) string {
	url_struct, err := url.Parse(downloadUrl)
	handleError(err)
	return filepath.Base(url_struct.Path)
}

func handleError(err error) {
	if err != nil {
		log.Println("err:", err)
		blockForWindows()
		os.Exit(1)
	}
}

func blockForWindows() { // Prevent windows from closing exe window.
	if runtime.GOOS == "windows" {
		for {
			log.Println("[Press `Ctrl+C` key to exit...]")
			time.Sleep(10 * time.Second)
		}
	}
}
