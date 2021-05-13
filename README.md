# golang-parallel-download-with-accept-ranges

Tested with `go1.16.3 linux/amd64`

# About

This is an example to show you how to download a file in efficient way.

If a URL supports http header - `Accept-Ranges`, it will be divided into several parts and download it concurrently.

![](https://github.com/jex-lin/golang-parallel-download-with-accept-ranges/blob/master/demo.gif)

## Build

1. Clone
    ```bash
    git clone https://github.com/unfor19/golang-parallel-download-with-accept-ranges.git && \
    cd golang-parallel-download-with-accept-ranges
    ```
1. Get dependencies
   ```bash
   go get
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