//go:build linux

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	log "github.com/cihub/seelog"
	. "github.com/wct-devops/image-transmit/core"
	"gopkg.in/yaml.v2"
)

func main() {
	a := app.New()
	InitI18nPrinter("")
	w := a.NewWindow(I18n.Sprintf("Image Transmit - WhaleCloud DevOps Team"))
	w.Resize(fyne.NewSize(900, 600))

	mw := &MyMainWindow{window: w}
	var loggerCfg []byte
	if _, err := os.Stat("logCfg.xml"); err == nil {
		loggerCfg, _ = ioutil.ReadFile("logCfg.xml")
	} else if _, err := os.Stat(filepath.Join(HOME, "logCfg.xml")); err == nil {
		loggerCfg, _ = ioutil.ReadFile(filepath.Join(HOME, "logCfg.xml"))
	}
	InitLogger(loggerCfg)

	CONF = new(YamlCfg)

	var cfgFile []byte
	_, err := os.Stat("cfg.yaml")
	if err != nil && os.IsNotExist(err) {
		_, err = os.Stat(filepath.Join(HOME, "cfg.yaml"))
		if err != nil && os.IsNotExist(err) {
			log.Error(I18n.Sprintf("Read cfg.yaml failed: %v", err))
		} else {
			cfgFile, err = ioutil.ReadFile(filepath.Join(HOME, "cfg.yaml"))
			if err != nil {
				log.Error(I18n.Sprintf("Read cfg.yaml failed: %v", err))
			}
		}
	} else {
		cfgFile, err = ioutil.ReadFile("cfg.yaml")
		if err != nil {
			log.Error(I18n.Sprintf("Read cfg.yaml failed: %v", err))
		}
	}

	err = yaml.Unmarshal(cfgFile, CONF)

	if len(CONF.Compressor) == 0 {
		if runtime.GOOS == "windows" {
			CONF.Compressor = "tar"
		} else {
			CONF.Compressor = "squashfs"
		}
	}

	if CONF.Compressor != "squashfs" {
		SQUASHFS = false
	}

	if len(CONF.Lang) > 1 {
		InitI18nPrinter(CONF.Lang)
	}

	if err != nil {
		w.SetContent(container.NewVBox(
			widget.NewLabelWithStyle(I18n.Sprintf("Configuration File Error"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel(fmt.Sprintf(I18n.Sprintf("Parse cfg.yaml file failed: %v, for instruction visit github.com/wct-devops/image-transmit"), err)),
		))
		w.ShowAndRun()
		return
	}

	if len(CONF.SrcRepos) < 1 || len(CONF.DstRepos) < 1 {
		w.SetContent(container.NewVBox(
			widget.NewLabelWithStyle(I18n.Sprintf("Configuration File Error"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel(I18n.Sprintf("Configuration File cfg.yaml incorrect, for instruction visit github.com/wct-devops/image-transmit")),
		))
		w.ShowAndRun()
		return
	}

	if CONF.Interval > 0 {
		INTERVAL = CONF.Interval
	}

	mw.compressor = CONF.Compressor
	mw.increment = false
	mw.singleFile = CONF.SingleFile
	mw.outputFormat = CONF.OutputFormat
	if mw.outputFormat == "" {
		mw.outputFormat = "both"
	}

	mw.srcRepo = &CONF.SrcRepos[0]
	mw.dstRepo = &CONF.DstRepos[0]

	// Source repo select
	srcNames := make([]string, len(CONF.SrcRepos))
	for i, v := range CONF.SrcRepos {
		srcNames[i] = repoDisplayName(v)
	}
	mw.cbSrcRepo = widget.NewSelect(srcNames, func(s string) {
		mw.srcRepo = findRepoByName(CONF.SrcRepos, s)
	})
	mw.cbSrcRepo.SetSelected(srcNames[0])

	// Destination repo select
	dstNames := make([]string, len(CONF.DstRepos))
	for i, v := range CONF.DstRepos {
		dstNames[i] = repoDisplayName(v)
	}
	mw.cbDstRepo = widget.NewSelect(dstNames, func(s string) {
		mw.dstRepo = findRepoByName(CONF.DstRepos, s)
	})
	mw.cbDstRepo.SetSelected(dstNames[0])

	// MaxConn
	mw.leMaxConn = widget.NewEntry()
	mw.leMaxConn.SetPlaceHolder("MaxThreads")
	if CONF.MaxConn > 0 {
		mw.leMaxConn.SetText(strconv.Itoa(CONF.MaxConn))
	} else {
		mw.leMaxConn.SetText(strconv.Itoa(runtime.NumCPU()))
	}

	// Retries
	mw.leRetries = widget.NewEntry()
	mw.leRetries.SetPlaceHolder("Retries")
	if CONF.Retries > 0 {
		mw.leRetries.SetText(strconv.Itoa(CONF.Retries))
	} else {
		mw.leRetries.SetText("2")
	}

	// Archive mode
	mw.cbIncrement = widget.NewSelect([]string{I18n.Sprintf("FULL"), I18n.Sprintf("INCR")}, func(s string) {
		mw.increment = s == I18n.Sprintf("INCR")
	})
	mw.cbIncrement.SetSelected(I18n.Sprintf("FULL"))

	// Single file
	mw.cbSingle = widget.NewSelect([]string{I18n.Sprintf("YES"), I18n.Sprintf("NO")}, func(s string) {
		mw.singleFile = s == I18n.Sprintf("YES")
	})
	if CONF.SingleFile {
		mw.cbSingle.SetSelected(I18n.Sprintf("YES"))
	} else {
		mw.cbSingle.SetSelected(I18n.Sprintf("NO"))
	}

	if SQUASHFS {
		mw.cbSingle.SetSelected(I18n.Sprintf("NO"))
		mw.cbSingle.Disable()
	}

	mw.cbOutputFmt = widget.NewSelect([]string{"OCI", "Docker", "Both"}, func(s string) {
		switch s {
		case "OCI":
			mw.outputFormat = "oci"
		case "Docker":
			mw.outputFormat = "docker"
		case "Both":
			mw.outputFormat = "both"
		}
		if mw.outputFormat == "oci" || mw.outputFormat == "docker" || mw.outputFormat == "both" {
			mw.cbSingle.SetSelected(I18n.Sprintf("NO"))
			mw.cbSingle.Disable()
		} else {
			if !SQUASHFS {
				mw.cbSingle.Enable()
			}
		}
	})
	switch mw.outputFormat {
	case "oci":
		mw.cbOutputFmt.SetSelected("OCI")
	case "docker":
		mw.cbOutputFmt.SetSelected("Docker")
	default:
		mw.cbOutputFmt.SetSelected("Both")
	}

	// Cache label
	mw.labelCache = widget.NewLabel(I18n.Sprintf("OFF"))

	// Buttons
	mw.btnSync = widget.NewButton(I18n.Sprintf("TRANSMIT"), func() { mw.Transmit() })
	mw.btnWatch = widget.NewButton(I18n.Sprintf("WATCH"), func() { mw.Watch() })
	mw.btnDownload = widget.NewButton(I18n.Sprintf("DOWNLOAD"), func() { mw.Download() })
	mw.btnUpload = widget.NewButton(I18n.Sprintf("UPLOAD"), func() { mw.Upload() })
	mw.btnCancel = widget.NewButton(I18n.Sprintf("CANCEL"), func() {
		mw.ctx.CancelFunc()
		mw.ctx.Info(I18n.Sprintf("User cancel it"))
	})
	mw.btnTest = widget.NewButton(I18n.Sprintf("VERIFY"), func() { mw.Verify() })

	// Status
	mw.labelStatus = widget.NewLabel("-----------")

	// Input area
	mw.teInput = widget.NewMultiLineEntry()
	mw.teInput.SetPlaceHolder(I18n.Sprintf("Image List (one image per line)"))

	// Output area
	mw.teOutput = widget.NewMultiLineEntry()
	mw.teOutput.Disable()

	// Layouts
	topRow := container.NewHBox(
		widget.NewLabel(I18n.Sprintf("Source:")),
		mw.cbSrcRepo,
		widget.NewLabel(I18n.Sprintf("Destination:")),
		mw.cbDstRepo,
		widget.NewLabel(I18n.Sprintf("MaxThreads:")),
		mw.leMaxConn,
		widget.NewLabel(I18n.Sprintf("Retries:")),
		mw.leRetries,
		widget.NewLabel(I18n.Sprintf("ArchiveMode:")),
		mw.cbIncrement,
		widget.NewLabel(I18n.Sprintf("SingleFile:")),
		mw.cbSingle,
		widget.NewLabel(I18n.Sprintf("OutputFormat:")),
		mw.cbOutputFmt,
		widget.NewLabel(I18n.Sprintf("LocalCache:")),
		mw.labelCache,
	)

	btnRow := container.NewHBox(
		mw.btnSync,
		mw.btnWatch,
		mw.btnDownload,
		mw.btnUpload,
		mw.btnCancel,
		mw.btnTest,
		widget.NewLabel(I18n.Sprintf("Status: ")),
		mw.labelStatus,
	)

	split := container.NewHSplit(
		container.NewBorder(
			widget.NewLabelWithStyle(I18n.Sprintf("Image List:"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			nil, nil, nil,
			mw.teInput,
		),
		container.NewBorder(
			widget.NewLabelWithStyle(I18n.Sprintf("Log Output:"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			nil, nil, nil,
			mw.teOutput,
		),
	)
	split.Offset = 0.4

	content := container.NewBorder(
		container.NewVBox(topRow, btnRow),
		nil, nil, nil,
		split,
	)

	w.SetContent(content)

	// Init context
	var lc *LocalCache
	if CONF.Cache.Pathname != "" {
		keepDays := 7
		keepSize := 10
		if CONF.Cache.KeepDays > 0 {
			keepDays = CONF.Cache.KeepDays
		}
		if CONF.Cache.KeepSize > 0 {
			keepSize = CONF.Cache.KeepSize
		}
		lc = NewLocalCache(CONF.Cache.Pathname, keepDays, keepSize)
		mw.labelCache.SetText(I18n.Sprintf("ON"))
	}

	lt := NewLocalTemp(TEMP_DIR)
	teLog := newGuiLogger(mw.teOutput)
	mw.ctx = NewTaskContext(teLog, lc, lt)

	if len(CONF.DingTalk) > 0 {
		mw.ctx.Notify = NewDingTalkWapper(CONF.DingTalk)
	}

	mw.ctx.Reset()

	// Status refresh goroutine
	go func() {
		for {
			time.Sleep(1 * time.Second)
			status := mw.ctx.GetStatus()
			fyne.Do(func() {
				mw.labelStatus.SetText(status)
			})
		}
	}()

	w.ShowAndRun()
}

func repoDisplayName(r Repo) string {
	if len(r.Name) > 0 {
		return r.Name
	}
	if r.Repository != "" {
		return r.Registry + "-" + r.Repository
	}
	return r.Registry
}

func findRepoByName(repos []Repo, name string) *Repo {
	for i, v := range repos {
		if repoDisplayName(v) == name {
			return &repos[i]
		}
	}
	return nil
}
