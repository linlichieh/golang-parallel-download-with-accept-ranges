# About

This is an example to show you how to download a file in efficient way.

If a URL supports http header - `Accept-Ranges`, it will be divided into several parts and download it concurrently.

# Run

Download a file with 5 connections (default: 5)

    ./golang-parallel-download-with-accept-ranges

File name with timestamp

    ./golang-parallel-download-with-accept-ranges -t

Specify the connection count

    ./golang-parallel-download-with-accept-ranges -c=7


![](https://github.com/jex-lin/golang-parallel-download-with-accept-ranges/blob/master/run.gif)

# Dependencies

[pb](github.com/cheggaaa/pb) - show multiple progress bar

    go get github.com/cheggaaa/pb

# Compile command

windows (64 bits)

    GOOS=windows GOARCH=amd64 go build -o download.exe

# TODO

* Support request header


