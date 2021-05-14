# ops

**Work In Progress**- An attempt to create an application that serves as a one-stop-shop for DevOps Engineers.

## Features

- Download a file using efficiently with [Accept-Ranges](https://developer.mozilla.org/en-US/docs/Web/HTTP/Range_requests)
- Unzip file

## Requirements

- [Go](https://golang.org/doc/install) v1.16.x

## Build

1. Clone
    ```bash
    git clone https://github.com/unfor19/ops.git && \
    cd op
    ```
1. Get dependencies
   ```bash
   go mod download
   ```
1. Go Build
   ```bash
   go build
   # output file: ./ops
   ```

## Usage

- Download a file with 5 connections (default: 5)
  ```bash
  ./ops
  ```
- File name with timestamp
  ```bash
  ./ops -t
  ```
- Specify the connection count
  ```bash
  ./ops -c=7
  ```

## Docker

1. Clone
    ```bash
    git clone https://github.com/unfor19/ops.git && \
    cd ops
    ```
1. Build
   ```bash
   docker build -t ops .
   ```
2. Run
   ```bash
   docker run --rm -it ops
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