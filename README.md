# golang-parallel-download-with-accept-ranges

Tested with `go1.16.3 linux/amd64`

Download a zip file in efficient way, and unzip it.

If a URL supports the http header [Accept-Ranges](https://developer.mozilla.org/en-US/docs/Web/HTTP/Range_requests), it will be divided into several parts and download it concurrently.

![](https://github.com/jex-lin/golang-parallel-download-with-accept-ranges/blob/master/demo.gif)

## Requirements

- [Go](https://golang.org/doc/install) v1.16.x

## Build

1. Clone
    ```bash
    git clone https://github.com/unfor19/golang-parallel-download-with-accept-ranges.git && \
    cd golang-parallel-download-with-accept-ranges
    ```
1. Get dependencies
   ```bash
   go mod download
   ```
1. Go Build
   ```bash
   go build
   # output file: ./golang-parallel-download-with-accept-ranges
   ```

## Usage

- Download a file with 5 connections (default: 5)
  ```bash
  ./golang-parallel-download-with-accept-ranges
  ```
- File name with timestamp
  ```bash
  ./golang-parallel-download-with-accept-ranges -t
  ```
- Specify the connection count
  ```bash
  ./golang-parallel-download-with-accept-ranges -c=7
  ```

## Docker

1. Clone
    ```bash
    git clone https://github.com/unfor19/golang-parallel-download-with-accept-ranges.git && \
    cd golang-parallel-download-with-accept-ranges
    ```
1. Build
   ```bash
   docker build -t ops .
   ```
1. Run **WIP**- need to fix `panic: runtime error: invalid memory address or nil pointer dereference, SIGSEGV`
   ```bash
   docker run --rm -t ops
   ```

## Dependencies

[pb](github.com/cheggaaa/pb) - show multiple progress bar

```bash
go get github.com/cheggaaa/pb
```

## Older README.md leftovers

<details>

<summary>Expand/Collapse</summary>

# Compile command

mac

    GOOS=darwin GOARCH=amd64 go build -o download.command

windows

    GOOS=windows GOARCH=amd64 go build -o download.exe

# FIXME

* File's body download on windows is different from one on mac. (e.g. mp4)

# TODO

* Support request header


</details>