package main

import (
	"fmt"
	"log"
	"strings"
	"net/http"
	"io/ioutil"
)

var gHtmlHead = `<!DOCTYPE html PUBLIC "-//WAPFORUM//DTD XHTML Mobile 1.0//EN" "http://www.wapforum.org/DTD/xhtml-mobile10.dtd">
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
<style type="text/css">
body{
    font-size:14px;
    line-height:22px;
    font-weight:bold;
    font-family:"Courier New", Verdana, Arial, Sans-serif;
}
div.stock {
    margin:2px 2px 16px 2px;
}
@media only screen and (max-device-width: 480px) {
    body {
        -webkit-text-size-adjust:100%%;
    }   
}
</style>
</head>
<body>
`
func homeHandler(w http.ResponseWriter, r *http.Request) {
    files, err := ioutil.ReadDir("./mf/")
    if err != nil {
	    fmt.Fprintf(w, "Internal Server Error")
	    return
    }

	fmt.Fprintf(w, gHtmlHead)
    fileCount := len(files)
    for i := 0; i < fileCount; i++ {
        file := files[fileCount - 1 - i]
	    fmt.Fprintf(w, "<a href=\"/mf/%s\">%s</a><br />\r\n", file.Name(),
	                strings.Replace(file.Name(), ".html", "", 1))
	    if strings.Index(file.Name(), "exbank") > 0 {
	          fmt.Fprintln(w, "<br />")
        }
    }
	fmt.Fprintln(w, "</body>\r\n</html>\r\n")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/mf/home", homeHandler)
	err := http.ListenAndServe(":11799", mux)
    log.Fatal(err)
}

