// +build windows

package main

import (
	"github.com/ArthurHlt/gominlog"
	"log"
	"github.com/fatih/color"
)

var logger *gominlog.MinLog = gominlog.NewMinLogWithWriter("cf-sync", gominlog.Linfo, true, log.Ldate | log.Ltime, color.Output)
