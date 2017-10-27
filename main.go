package main

import (
	"flag"
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
	Url       string
	File      *os.File
	Count     int
	SyncWG    sync.WaitGroup
	TotalSize int
	Progress
}

type Progress struct {
	Bars   *multibar.BarContainer
	Update map[int]multibar.ProgressFunc
}

func main() {
	var t = flag.Bool("t", false, "file name with datetime")
	var worker_count = flag.Int("c", 5, "connection count")
	flag.Parse()

	var download_url string
	fmt.Print("Please enter a url: ")
	fmt.Scanf("%s", &download_url)

	// Get header from the url
	log.Printf("Url: %s\n", download_url)
	total_size := getSizeAndCheckRangeSupport(download_url)
	log.Printf("File size: %d bytes\n", total_size)

	var file_path string
	if *t {
		file_path = filepath.Dir(os.Args[0]) + "/" + strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + getFileName(download_url)
	} else {
		file_path = filepath.Dir(os.Args[0]) + "/" + getFileName(download_url)
	}
	log.Printf("Local path: %s\n", file_path)
	f, err := os.OpenFile(file_path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal("Failed to create file, error:", err)
	}
	defer f.Close()

	// New worker struct for downloading file
	var worker = Worker{
		Url:       download_url,
		File:      f,
		Count:     *worker_count,
		TotalSize: total_size,
	}

	// Progress bar
	worker.Progress.Bars, _ = multibar.New()
	worker.Progress.Update = make(map[int]multibar.ProgressFunc)

	var start, end int
	var partial_size = int(total_size / *worker_count)

	for num := 1; num <= worker.Count; num++ {
		// Print progress bar
		worker.Progress.Update[num] = worker.Progress.Bars.MakeBar(100, fmt.Sprintf("Part %d", num))

		if num == worker.Count {
			end = total_size // last part
		} else {
			end = start + partial_size
		}

		worker.SyncWG.Add(1)
		go worker.writeRange(num, start, end-1)
		start = end
	}
	go worker.Progress.Bars.Listen()
	worker.SyncWG.Wait()
	time.Sleep(300 * time.Millisecond) // Wait for progress bar UI to be done.
	log.Println("Done!")
}

func (w *Worker) writeRange(part_num int, start int, end int) {
	defer w.SyncWG.Done()
	var written int
	body, size, err := w.getRangeBody(part_num, start, end)
	if err != nil {
		log.Fatalf("Part %d request error: %s\n", part_num, err.Error())
	}
	defer body.Close()

	percent_flag := map[int]bool{}
	buf := make([]byte, 32*1024) // make a buffer to keep chunks that are read
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

			// Report progress and only report once time by every 1%.
			p := int(float32(written) / float32(size) * 100)
			_, flagged := percent_flag[p]
			if !flagged {
				percent_flag[p] = true
				w.Progress.Update[part_num](p)
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
			log.Fatalf("Part %d occured error: %s\n", part_num, er.Error())
		}
	}
}

func (w *Worker) getRangeBody(part_num int, start int, end int) (io.ReadCloser, int, error) {
	var client http.Client
	req, err := http.NewRequest("GET", w.Url, nil)
	// req.Header.Set("cookie", "")
	if err != nil {
		return nil, 0, err
	}

	// Set range header
	req.Header.Add("Range", "bytes="+strconv.Itoa(start)+"-"+strconv.Itoa(end))
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	size, _ := strconv.Atoi(resp.Header["Content-Length"][0])
	return resp.Body, size, err
}

func getSizeAndCheckRangeSupport(url string) (size int) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	// req.Header.Set("cookie", "")
	log.Printf("Request header: %s\n", req.Header)
	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Printf("Response header: %v\n", res.Header)
	header := res.Header
	accept_ranges, supported := header["Accept-Ranges"]
	if !supported {
		log.Fatal("Doesn't support `Accept-Ranges`.")
	} else if supported && accept_ranges[0] != "bytes" {
		log.Fatal("Support `Accept-Ranges`, but value is not `bytes`.")
	}
	size, _ = strconv.Atoi(header["Content-Length"][0]) // Get the content length.
	return
}

func getFileName(download_url string) string {
	url_struct, _ := url.Parse(download_url)
	return filepath.Base(url_struct.Path)
}
