# About

This is a example to show you how to download a file in efficient way.

If the url supports http header `Accept-Ranges`, it will divide it into 5 parts and download it concurrently.

# Run

Download a file

    go run main.go "http://ipv4.download.thinkbroadband.com/20MB.zip"

![](https://github.com/jex-lin/golang-parallel-download-with-accept-ranges/blob/master/run.gif)

# Dependencies

sethgrid/multibar - show multiple progress bar.

    go get github.com/sethgrid/multibar


