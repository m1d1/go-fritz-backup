package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/m1d1/go-fritz-backup/ctrl"
)

var (
	Version   string = "<development Version>"
	BuildDate string = "N/A"
)

func main() {
	start := time.Now()
	fmt.Printf("go-fritz-backup %s (%s build %s)\nMichael Dinkelaker 2023-2025\n\n", Version, runtime.Version(), BuildDate)

	c := ctrl.New()

	if err := c.ReadConfigFile(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	deviceInfo, err := c.GetDeviceInfo()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(2)
	}
	fmt.Println(deviceInfo)

	// Core function - Backup configfile - this takes a bit longer
	downloadUrl, fileName, url, err := c.GetConfigFile()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(3)
	}

	cnfMsg, err := c.BackupConfigFile(url, downloadUrl, fileName)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(4)
	}
	fmt.Println(cnfMsg)

	if c.Settings.Export.Phonebooks {
		pbMsg, err := c.BackupPhonebooks()
		if err != nil {
			slog.Error(err.Error())
			os.Exit(5)
		}
		fmt.Printf(pbMsg)
	}

	if c.Settings.Export.PhoneBarringList {
		blMsg, err := c.DownloadCallBarringList()
		if err != nil {
			slog.Error(err.Error())
			os.Exit(6)
		}
		fmt.Println(blMsg)
	}

	if c.Settings.Export.PhoneAssets {
		aMsg, err := c.GetAssetsFile()
		if err != nil {
			slog.Error(err.Error())
			os.Exit(7)
		}
		fmt.Println(aMsg)
	}
	fmt.Printf("finished in %s\n", time.Since(start))
}
