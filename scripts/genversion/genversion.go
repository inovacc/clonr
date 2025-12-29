//go:build ignore

package main

import (
	"github.com/inovacc/genversioninfo"
)

func main() {
	opts := genversioninfo.Options{
		GoVersionPath:    ".go-version",
		PlainVersionPath: "internal/version/version.go",
	}

	if err := genversioninfo.GenVersionInfo(opts); err != nil {
		panic(err)
	}
}
