package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/sethgrid/multibar"
)

type Worker struct {
	Url          string
	File         *os.File
	Count        int
	SyncWG       sync.WaitGroup
	TotalSize    int
	ProgressBars map[int]multibar.ProgressFunc
}

func main() {
	// e.g. http://ipv4.download.thinkbroadband.com/20MB.zip
	// e.g. http://ipv4.download.thinkbroadband.com/50MB.zip
	if len(os.Args) == 1 {
		log.Fatal("Please pass url.")
	}
	download_url := os.Args[1]
	worker_count := 5 // Goroutine number

	// Get header
	res, _ := http.Head(download_url)
	header := res.Header

	accept_ranges, supported := header["Accept-Ranges"]
	if !supported {
		log.Fatal("Doesn't support `Accept-Ranges`.")
	} else if accept_ranges[0] != "bytes" {
		log.Fatal("Support `Accept-Ranges`, but not equal to `bytes`.")
	}

	f, err := os.OpenFile(getFileName(download_url), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal("Failed to create file, error:", err)
	}
	defer f.Close()

	total_size, _ := strconv.Atoi(header["Content-Length"][0]) // Get the content length.
	partial_size := int(total_size / worker_count)

	var worker = Worker{
		Url:       download_url,
		File:      f,
		Count:     worker_count,
		TotalSize: total_size,
	}

	fmt.Println("Total size:", worker.TotalSize)
	fmt.Println("Downloading ..., please wait.")

	// Show progress bar
	progressBars, _ := multibar.New()
	worker.ProgressBars = make(map[int]multibar.ProgressFunc)

	var start, end int
	for num := 1; num <= worker.Count; num++ {
		worker.ProgressBars[num] = progressBars.MakeBar(100, fmt.Sprintf("Part %d", num))

		if num == worker.Count {
			end = total_size // last part
		} else {
			end = start + partial_size
		}

		worker.SyncWG.Add(1)
		go worker.writeRange(num, start, end-1)
		start = end
	}
	go progressBars.Listen()
	worker.SyncWG.Wait()
	time.Sleep(300 * time.Millisecond) // Wait for progress bar UI to be done.
	fmt.Println("Done!")
}

func (w *Worker) writeRange(part_num int, start int, end int) {
	defer w.SyncWG.Done()
	var written int
	body, size, err := w.getRangeBody(part_num, start, end)
	if err != nil {
		log.Fatalf("Part %d request error: %s\n", part_num, err.Error())
	}
	defer body.Close()

	percent_flag := map[int]bool{} // Prevent reporting repeatedly.
	buf := make([]byte, 2*1024)    // make a buffer to keep chunks that are read
	for {
		nr, er := body.Read(buf)
		if nr > 0 {
			nw, err := w.File.WriteAt(buf[0:nr], int64(start))
			if err != nil {
				log.Fatalf("Part %d occured error: %s.\n", part_num, err.Error())
			}
			if nr != nw {
				log.Fatalf("Part %d occured error of short writiing.\n", part_num)
			}

			start = int(nw) + start
			if nw > 0 {
				written += nw
			}

			p := int(float32(written) / float32(size) * 100)

			// Report progress and only report once time by every 1%.
			_, flagged := percent_flag[p]
			if p%1 == 0 && !flagged {
				percent_flag[p] = true
				w.ProgressBars[part_num](p)
			}
		}
		if er != nil {
			if er.Error() == "EOF" {
				if size == written {
					// Downloading successfully
				} else {
					log.Fatalf("Part %d unfinished.\n", part_num)
				}
				break
			}
			log.Fatal("Part %d occured error: %s", part_num, er.Error())
		}
	}
}

func (w *Worker) getRangeBody(part_num int, start int, end int) (io.ReadCloser, int, error) {
	var client http.Client
	req, err := http.NewRequest("GET", w.Url, nil)
	if err != nil {
		return nil, 0, err
	}

	// Set range header
	req.Header.Add("Range", "bytes="+strconv.Itoa(start)+"-"+strconv.Itoa(end))
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusPartialContent {
		return nil, 0, errors.New("Accept-Ranges not supported.")
	}
	size, _ := strconv.Atoi(resp.Header["Content-Length"][0])
	return resp.Body, size, err
}

func getFileName(download_url string) string {
	url_struct, _ := url.Parse(download_url)
	return filepath.Base(url_struct.Path)
}
