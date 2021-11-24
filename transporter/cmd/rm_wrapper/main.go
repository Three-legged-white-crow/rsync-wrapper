package main

import "flag"

const emptyValue = "empty"

func main() {
	path := flag.String("path", emptyValue, "abs path that remove")
	isExcludeSrcDir := flag.Bool("exclude-dir", false, "if specify path is dir, exclude dir or not")
	flag.Parse()

	
}
