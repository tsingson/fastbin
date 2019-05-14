package main

import (
	"github.com/tsingson/fastbin"
	"github.com/tsingson/fastbin/example/fb_types/module"
)

func main() {
	fastbin.Register(&module.MyStruct{})
	fastbin.GenCode()
}
