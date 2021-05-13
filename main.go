// Based on: https://github.com/jacklin293/golang-parallel-download-with-accept-ranges
package main

import (
	"archive/zip"
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
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb"
)

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

func download(download_url string) {
	var t = flag.Bool("t", false, "file name with datetime")
	var worker_count = flag.Int64("c", 5, "connection count")
	flag.Parse()

	// Get header from the url
	log.Println("Url:", download_url)
	file_size, err := getSizeAndCheckRangeSupport(download_url)
	handleError(err)
	log.Printf("File size: %d Bytes\n", file_size)

	var file_path string
	if *t {
		file_path = filepath.Dir(os.Args[0]) + string(filepath.Separator) + strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + getFileName(download_url)
	} else {
		file_path = filepath.Dir(os.Args[0]) + string(filepath.Separator) + getFileName(download_url)
	}
	log.Printf("Local path: %s\n", file_path)

	if _, err := os.Stat(file_path); err == nil {
		log.Printf("Zip file exists, removing ...")
		os.Remove(file_path)
		log.Printf("Successfully removed Zip file")
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

	// New worker struct to download file
	var worker = Worker{
		Url:       download_url,
		File:      f,
		Count:     *worker_count,
		TotalSize: file_size,
	}

	var start, end int64
	var partial_size = int64(file_size / *worker_count)
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

		if num == worker.Count {
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

func getSizeAndCheckRangeSupport(url string) (size int64, err error) {
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
	accept_ranges, supported := header["Accept-Ranges"]
	if !supported {
		return 0, errors.New("Doesn't support header `Accept-Ranges`.")
	} else if supported && accept_ranges[0] != "bytes" {
		return 0, errors.New("Support `Accept-Ranges`, but value is not `bytes`.")
	}
	size, err = strconv.ParseInt(header["Content-Length"][0], 10, 64)
	return
}

func getFileName(download_url string) string {
	url_struct, err := url.Parse(download_url)
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

// Source: https://stackoverflow.com/a/24792688/5285732
func Unzip(src string, dest string) error {
	// src - zip file
	// dest -  auto creates target directory and extracts the files to it
	r, err := zip.OpenReader(src)
	if err != nil {
		log.Println("Failed to open source file", src)
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Println("Failed to close file", src)
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			log.Println("Failed to open output file", f.Name)
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				log.Println("Failed to close output file", f.Name)
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					log.Println("Failed to close file", f.Name())
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				log.Println("Failed to copy file", f.Name())
				log.Println(err)
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			log.Println("Failed to extract and write file", f.Name)
			return err
		}
	}

	return nil
}

func main() {

	var download_url string
	// fmt.Print("Please enter a URL: ")
	// fmt.Scanf("%s", &download_url)

	if download_url == "" {
		download_url = "https://releases.hashicorp.com/terraform/0.15.3/terraform_0.15.3_linux_amd64.zip"
	}

	var file_name = getFileName(download_url)
	current_dir, err := os.Getwd()
	if err != nil {
		return
	}
	var file_path = current_dir + string(filepath.Separator) + file_name
	var dest_dir = current_dir + string(filepath.Separator) + "terraform"

	download(download_url)
	log.Println("Unzipping", file_path, "to", dest_dir)

	unzip_error := Unzip(file_path, dest_dir)
	if unzip_error != nil {
		log.Fatal(unzip_error)
	}
	log.Println("Successfully unzipped", file_name, "to", dest_dir)

}
