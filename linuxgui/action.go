//go:build linux

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	log "github.com/cihub/seelog"
	"github.com/mcuadros/go-version"
	. "github.com/wct-devops/image-transmit/core"
	"gopkg.in/yaml.v2"
)

type MyMainWindow struct {
	window       fyne.Window
	teInput      *widget.Entry
	teOutput     *widget.Entry
	btnSync      *widget.Button
	btnWatch     *widget.Button
	btnCancel    *widget.Button
	btnDownload  *widget.Button
	btnUpload    *widget.Button
	btnTest      *widget.Button
	cbSrcRepo    *widget.Select
	cbDstRepo    *widget.Select
	cbIncrement  *widget.Select
	cbSingle     *widget.Select
	cbOutputFmt  *widget.Select
	labelStatus  *widget.Label
	labelCache   *widget.Label
	leMaxConn    *widget.Entry
	leRetries    *widget.Entry
	srcRepo      *Repo
	dstRepo      *Repo
	ctx          *TaskContext
	cmUpload     *CompressionMetadata
	pathUpload   string
	maxConn      int
	retries      int
	compressor   string
	singleFile   bool
	increment    bool
	outputFormat string
}

func (mw *MyMainWindow) BeginAction() bool {
	txtLen := len(mw.teOutput.Text)
	if txtLen > 1000000 {
		mw.teOutput.SetText(mw.teOutput.Text[txtLen-1000000:])
	}
	maxConn, err := strconv.Atoi(mw.leMaxConn.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf(I18n.Sprintf("Failed to set 'MaxThreads' with error: %v", err)), mw.window)
		return false
	}
	mw.maxConn = maxConn

	retries, err := strconv.Atoi(mw.leRetries.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf(I18n.Sprintf("Failed to set 'Retries' with error: %v", err)), mw.window)
		return false
	}
	mw.retries = retries

	mw.ctx.Info(I18n.Sprintf("==============BEGIN=============="))
	mw.ctx.Info(I18n.Sprintf("Transmit params: max threads: %v, max retries: %v", mw.maxConn, retries))
	mw.ctx.Reset()
	mw.ctx.UpdateSecStart(time.Now().Unix())
	return true
}

func (mw *MyMainWindow) EndAction() {
	if !CONF.KeepTemp {
		mw.ctx.Temp.Clean()
	}
	mw.ctx.UpdateSecEnd(time.Now().Unix())
	if mw.ctx.Notify != nil {
		mw.ctx.Notify.Send(I18n.Sprintf("### Transmit Task End \n- Stat: %v", mw.ctx.GetStatus()))
	}
	mw.ctx.Info(I18n.Sprintf("===============END==============="))
	log.Flush()
}

func (mw *MyMainWindow) StatDatafiles(pathname string, filename string) error {
	for k := range mw.ctx.CompMeta.Datafiles {
		i, err := os.Stat(filepath.Join(pathname, k))
		if err != nil {
			return mw.ctx.Errorf(I18n.Sprintf("Stat data file failed: %v", err))
		}
		mw.ctx.CompMeta.AddDatafile(k, i.Size())
	}
	b, err := yaml.Marshal(mw.ctx.CompMeta)
	if err != nil {
		return mw.ctx.Errorf(I18n.Sprintf("Save meta yaml file failed: %v", err))
	}
	metaFile := filepath.Join(pathname, filename+"_meta.yaml")
	err = ioutil.WriteFile(metaFile, b, os.ModePerm)
	if err != nil {
		return mw.ctx.Errorf(I18n.Sprintf("Save meta yaml file failed: %v", err))
	}
	mw.ctx.Info(I18n.Sprintf("Create meta file: %s", metaFile))
	return nil
}

func (mw *MyMainWindow) getInputList() []string {
	var list []string
	input := mw.teInput.Text
	input = strings.ReplaceAll(input, "\t", "")

	if CheckInvalidChar(strings.ReplaceAll(strings.ReplaceAll(input, "\r", ""), "\n", "")) {
		dialog.ShowError(fmt.Errorf(I18n.Sprintf("Invalid char(s) found from the input, please check the text in the left edit box")), mw.window)
		return nil
	}
	imgList := strings.Split(strings.ReplaceAll(input, "\r", ""), "\n")
	for _, imgName := range imgList {
		imgName = strings.TrimSpace(imgName)
		if imgName == "" {
			continue
		}
		list = append(list, imgName)
	}
	if len(list) < 1 {
		dialog.ShowError(fmt.Errorf(I18n.Sprintf("The image list is empty, please input on the left edit box, one image each line")), mw.window)
		return nil
	}
	return list
}

func (mw *MyMainWindow) Transmit() {
	imgList := mw.getInputList()
	if imgList == nil || len(imgList) < 1 || !mw.BeginAction() {
		return
	}

	c, err := NewClient(mw.maxConn, mw.retries, mw.ctx)
	if err != nil {
		dialog.ShowError(err, mw.window)
		return
	}

	go func() {
		fyne.Do(func() {
			mw.btnSync.Disable()
		})
		defer fyne.Do(func() {
			mw.btnSync.Enable()
		})
		for _, rawURL := range imgList {
			if mw.ctx.Cancel() {
				mw.ctx.Errorf(I18n.Sprintf("User cancelled..."))
				return
			}
			src, dst := GenRepoUrl(mw.srcRepo.Registry, mw.dstRepo.Registry, mw.dstRepo.Repository, rawURL)
			c.GenerateOnlineTask(src, mw.srcRepo, dst, mw.dstRepo)
		}
		mw.ctx.UpdateTotalTask(c.TaskLen())
		c.Run()
		mw.EndAction()
	}()
}

func (mw *MyMainWindow) Watch() {
	imgList := mw.getInputList()
	if imgList == nil || len(imgList) < 1 || !mw.BeginAction() {
		return
	}

	c, err := NewClient(mw.maxConn, mw.retries, mw.ctx)
	if err != nil {
		dialog.ShowError(err, mw.window)
		return
	}

	go func() {
		fyne.Do(func() {
			mw.btnWatch.Disable()
		})
		defer func() {
			mw.EndAction()
			fyne.Do(func() {
				mw.btnWatch.Enable()
			})
			mw.ctx.History = nil
		}()

		mw.ctx.History, err = NewHistory(HIS_FILE)
		if err != nil {
			log.Error(err)
			return
		}

		for {
			for _, rawURL := range imgList {
				if mw.ctx.Cancel() {
					mw.ctx.Errorf(I18n.Sprintf("User cancelled..."))
					return
				}

				src, dst := GenRepoUrl(mw.srcRepo.Registry, mw.dstRepo.Registry, mw.dstRepo.Repository, rawURL)
				srcURL, _ := NewRepoURL(src)
				dstURL, _ := NewRepoURL(dst)

				imageSourceSrc, err := NewImageSource(mw.ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), "", mw.srcRepo.User, mw.srcRepo.Password, !strings.HasPrefix(src, "https") || mw.srcRepo.Insecure)
				if err != nil {
					log.Error(err)
					return
				}

				tags, err := imageSourceSrc.GetSourceRepoTags()
				if err != nil {
					c.PutAInvalidTask(src)
					mw.ctx.Error(I18n.Sprintf("Fetch tag list failed for %v with error: %v", srcURL, err))
					continue
				}

				for _, tag := range tags {
					newSrcUrl := srcURL.GetRegistry() + "/" + srcURL.GetRepoWithNamespace() + ":" + tag
					newDstUrl := dstURL.GetRegistry() + "/" + dstURL.GetRepoWithNamespace() + ":" + tag
					if version.Compare(tag, srcURL.GetTag(), "<") || mw.ctx.History.Skip(newSrcUrl) {
						continue
					}

					newImgSrc, err := NewImageSource(mw.ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), tag, mw.srcRepo.User, mw.srcRepo.Password, !strings.HasPrefix(newSrcUrl, "https") || mw.srcRepo.Insecure)
					if err != nil {
						c.PutAInvalidTask(newSrcUrl)
						mw.ctx.Error(I18n.Sprintf("Url %s format error: %v, skipped", newSrcUrl, err))
						continue
					}

					newImgDst, err := NewImageDestination(mw.ctx.Context, dstURL.GetRegistry(), dstURL.GetRepoWithNamespace(), tag, mw.dstRepo.User, mw.dstRepo.Password, !strings.HasPrefix(newDstUrl, "https") || mw.dstRepo.Insecure)
					if err != nil {
						c.PutAInvalidTask(newSrcUrl)
						mw.ctx.Error(I18n.Sprintf("Url %s format error: %v, skipped", newDstUrl, err))
						continue
					}
					var callback func(bool, string)
					if mw.ctx.Notify != nil {
						callback = func(result bool, content string) {
							if result {
								mw.ctx.Notify.Send(I18n.Sprintf("### Transmit Success\n- Image: %v\n- Stat: %v", newDstUrl, content))
							} else {
								mw.ctx.Notify.Send(I18n.Sprintf("### Transmit Failed\n- Image: %v\n- Error: %v", newDstUrl, content))
							}
						}
					}
					c.PutATask(NewOnlineTaskCallback(newImgSrc, newImgDst, mw.ctx, callback))
					mw.ctx.Info(I18n.Sprintf("Generated a task for %s to %s", newSrcUrl, newDstUrl))
				}
				imageSourceSrc.Close()
			}
			mw.ctx.UpdateTotalTask(mw.ctx.GetTotalTask() + c.TaskLen())
			c.Run()
			select {
			case <-mw.ctx.Context.Done():
				mw.ctx.Errorf(I18n.Sprintf("User cancelled..."))
				return
			case <-time.After(time.Duration(INTERVAL) * time.Second):
				c.ClearInvalidTask()
				continue
			}
		}
	}()
}

func (mw *MyMainWindow) Download() {
	imgList := mw.getInputList()
	if imgList == nil || len(imgList) < 1 || !mw.BeginAction() {
		return
	}

	if mw.maxConn > len(imgList) {
		mw.maxConn = len(imgList)
	}
	c, err := NewClient(mw.maxConn, mw.retries, mw.ctx)
	if err != nil {
		dialog.ShowError(err, mw.window)
		return
	}

	var prefixPathname string
	var prefixFilename string
	if len(CONF.OutPrefix) > 0 {
		prefixPathIdx := strings.LastIndex(CONF.OutPrefix, string(os.PathSeparator))
		if prefixPathIdx > 0 {
			prefixPathname = CONF.OutPrefix[0:prefixPathIdx]
			prefixFilename = CONF.OutPrefix[prefixPathIdx+1:]
		}
	}

	pathname := filepath.Join(HOME, time.Now().Format("20060102"), prefixPathname)
	_, err = os.Stat(pathname)
	if os.IsNotExist(err) {
		os.MkdirAll(pathname, os.ModePerm)
	}

	var workName string
	if mw.increment {
		workName = time.Now().Format("img_incr_200601021504")
	} else {
		workName = time.Now().Format("img_full_200601021504")
	}
	if len(prefixFilename) > 0 {
		workName = prefixFilename + "_" + workName
	}

	mw.ctx.CreateCompressionMetadata(mw.compressor)

	// Load local blob store for automatic incremental
	if CONF.LocalStore != "" {
		localStorePath := CONF.LocalStore
		if storeInfo, statErr := os.Stat(localStorePath); statErr == nil && storeInfo.IsDir() {
			if localStore, lsErr := NewOCILayout(localStorePath); lsErr == nil {
				blobs, _ := localStore.ListBlobs()
				for _, b := range blobs {
					mw.ctx.CompMeta.BlobDone(b.Hex(), "local-store")
				}
				mw.ctx.Info(I18n.Sprintf("Loaded %d blobs from local store: %s", len(blobs), localStorePath))
			}
		}
	}

	if mw.increment {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()
			incPath := reader.URI().Path()
			mw.ctx.Info(I18n.Sprintf("Selected the history image file: %s", incPath))

			if mw.outputFormat == "oci" || mw.outputFormat == "docker" || mw.outputFormat == "both" {
				incInfo, statErr := os.Stat(incPath)
				if statErr == nil && incInfo.IsDir() {
					incOli, ociErr := NewOCILayout(incPath)
					if ociErr == nil {
						blobs, _ := incOli.ListBlobs()
						for _, b := range blobs {
							mw.ctx.CompMeta.BlobDone(b.Hex(), fmt.Sprintf("oci-layout:%s", incPath))
						}
						mw.downloadRun(c, pathname, workName, imgList, &incPath)
						return
					}
				}
				if IsOCILayoutTar(incPath) {
					tmpExtractDir := filepath.Join(TEMP_DIR, "oci-inc-extract")
					incOli, ociErr := ExtractFromTar(incPath, tmpExtractDir)
					if ociErr == nil {
						blobs, _ := incOli.ListBlobs()
						for _, b := range blobs {
							mw.ctx.CompMeta.BlobDone(b.Hex(), fmt.Sprintf("oci-layout-tar:%s", incPath))
						}
						mw.downloadRun(c, pathname, workName, imgList, &incPath)
						return
					}
				}
			}

			b, readErr := ioutil.ReadFile(incPath)
			if readErr != nil {
				dialog.ShowError(fmt.Errorf(I18n.Sprintf("Open file failed: %v", readErr)), mw.window)
				return
			}
			cm := new(CompressionMetadata)
			err = yaml.Unmarshal(b, cm)
			if err != nil {
				dialog.ShowError(fmt.Errorf(I18n.Sprintf("Parse file failed(version incompatible or file corrupt?): %v", err)), mw.window)
				return
			}
			for k := range cm.Blobs {
				mw.ctx.CompMeta.BlobDone(k, fmt.Sprintf("https://last.img/skip/it:%s", filepath.Base(incPath)))
			}
			mw.downloadRun(c, pathname, workName, imgList, nil)
		}, mw.window)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".yaml", ".tar", ".tgz"}))
		fd.Show()
		return
	}

	mw.downloadRun(c, pathname, workName, imgList, nil)
}

func (mw *MyMainWindow) downloadRun(c *Client, pathname string, workName string, imgList []string, incOCIPath *string) {
	var ociPath string
	if mw.outputFormat == "oci" || mw.outputFormat == "docker" || mw.outputFormat == "both" {
		ociPath = filepath.Join(pathname, workName)
		if err := mw.ctx.CreateOCILayout(ociPath); err != nil {
			dialog.ShowError(fmt.Errorf(I18n.Sprintf("Create OCI layout failed: %v", err)), mw.window)
			return
		}
		mw.ctx.Info(I18n.Sprintf("Create OCI layout: %s", ociPath))

		if incOCIPath != nil {
			incPath := *incOCIPath
			incInfo, statErr := os.Stat(incPath)
			if statErr == nil && incInfo.IsDir() {
				incOli, ociErr := NewOCILayout(incPath)
				if ociErr == nil {
					blobs, _ := incOli.ListBlobs()
					for _, b := range blobs {
						mw.ctx.CompMeta.BlobDone(b.Hex(), fmt.Sprintf("oci-layout:%s", incPath))
					}
				}
			} else if IsOCILayoutTar(incPath) {
				tmpExtractDir := filepath.Join(TEMP_DIR, "oci-inc-extract-run")
				incOli, ociErr := ExtractFromTar(incPath, tmpExtractDir)
				if ociErr == nil {
					blobs, _ := incOli.ListBlobs()
					for _, b := range blobs {
						mw.ctx.CompMeta.BlobDone(b.Hex(), fmt.Sprintf("oci-layout-tar:%s", incPath))
					}
				}
			}
		}
	} else if SQUASHFS {
		mw.ctx.Temp.SavePath(workName)
		mw.ctx.CreateSquashfsTar(TEMP_DIR, workName, "")
	} else {
		if mw.singleFile {
			mw.ctx.CreateSingleWriter(pathname, workName, mw.compressor)
		} else {
			mw.ctx.CreateTarWriter(pathname, workName, mw.compressor, mw.maxConn)
		}
	}

	go func() {
		fyne.Do(func() {
			mw.btnDownload.Disable()
		})
		for _, rawURL := range imgList {
			if mw.ctx.Cancel() {
				mw.ctx.Errorf(I18n.Sprintf("User cancelled..."))
				return
			}
			src, _ := GenRepoUrl(mw.srcRepo.Registry, mw.dstRepo.Registry, mw.dstRepo.Repository, rawURL)
			c.GenerateOfflineDownTask(src, rawURL, mw.srcRepo)
		}
		mw.ctx.UpdateTotalTask(c.TaskLen())
		c.Run()
		if mw.ctx.OCILayout != nil {
			if err := mw.ctx.OCILayout.SaveIndex(); err != nil {
				mw.ctx.Error(I18n.Sprintf("Save OCI index failed: %v", err))
			}
			mw.ctx.Info(I18n.Sprintf("OCI layout saved to %s", ociPath))

			// Update local blob store with new blobs
			if CONF.LocalStore != "" {
				localStorePath := CONF.LocalStore
				if _, statErr := os.Stat(localStorePath); os.IsNotExist(statErr) {
					os.MkdirAll(localStorePath, os.ModePerm)
				}
				if localStore, lsErr := NewOCILayout(localStorePath); lsErr == nil {
					if copyErr := localStore.CopyBlobsFrom(mw.ctx.OCILayout); copyErr != nil {
						mw.ctx.Info(I18n.Sprintf("Update local store failed: %v", copyErr))
					} else {
						mw.ctx.Info(I18n.Sprintf("Local blob store updated: %s", localStorePath))
					}
				}
			}

			tarName := workName + ".tar"
			tarPath := filepath.Join(pathname, tarName)
			var localStore *OCILayout
			if CONF.LocalStore != "" {
				localStore, _ = NewOCILayout(CONF.LocalStore)
			}
			switch mw.outputFormat {
			case "oci":
				mw.ctx.Info(I18n.Sprintf("Packing OCI tar: %s", tarPath))
				if err := mw.ctx.OCILayout.PackToTar(tarPath); err != nil {
					mw.ctx.Error(I18n.Sprintf("Pack tar failed: %v", err))
				} else {
					mw.ctx.Info(I18n.Sprintf("OCI tar created: %s", tarPath))
				}
			case "docker":
				mw.ctx.Info(I18n.Sprintf("Packing Docker tar: %s", tarPath))
				if err := mw.ctx.OCILayout.PackToDockerTar(tarPath, localStore, true); err != nil {
					mw.ctx.Error(I18n.Sprintf("Pack tar failed: %v", err))
				} else {
					mw.ctx.Info(I18n.Sprintf("Docker tar created: %s", tarPath))
				}
			default: // "both" or ""
				mw.ctx.Info(I18n.Sprintf("Packing OCI+Docker tar: %s", tarPath))
				if err := mw.ctx.OCILayout.PackToDockerTar(tarPath, localStore, false); err != nil {
					mw.ctx.Error(I18n.Sprintf("Pack tar failed: %v", err))
				} else {
					mw.ctx.Info(I18n.Sprintf("OCI+Docker tar created: %s", tarPath))
				}
			}
			mw.EndAction()
			fyne.Do(func() {
				mw.btnDownload.Enable()
			})
		} else if mw.ctx.SingleWriter != nil {
			time.Sleep(1 * time.Second)
			mw.ctx.SingleWriter.SetQuit()
		} else if mw.ctx.TarWriter != nil {
			mw.ctx.CloseTarWriter()
		}

		if mw.ctx.OCILayout == nil && mw.ctx.SingleWriter == nil {
			if mw.ctx.SquashfsTar != nil {
				mw.ctx.Info(I18n.Sprintf("Mksquashfs Compress Start"))
				err := MakeSquashfs(mw.ctx.GetLogger(), filepath.Join(TEMP_DIR, workName), filepath.Join(pathname, workName+".squashfs"))
				mw.ctx.Info(I18n.Sprintf("Mksquashfs Compress End"))
				if err != nil {
					mw.ctx.Error(I18n.Sprintf("Mksquashfs compress failed with %v", err))
					fyne.Do(func() {
						dialog.ShowError(fmt.Errorf(I18n.Sprintf("Mksquashfs compress failed with %v", err)), mw.window)
					})
					return
				}
				mw.ctx.CompMeta.AddDatafile(workName+".squashfs", 0)
			}
			mw.StatDatafiles(pathname, workName)
			mw.EndAction()
			fyne.Do(func() {
				mw.btnDownload.Enable()
			})
		}
	}()
	if mw.ctx.SingleWriter != nil {
		go func() {
			mw.ctx.SingleWriter.Run()
			mw.ctx.SingleWriter.SaveDockerMeta(mw.ctx.CompMeta)
			mw.StatDatafiles(pathname, workName)
			mw.EndAction()
			fyne.Do(func() {
				mw.btnDownload.Enable()
			})
		}()
	}
}

func (mw *MyMainWindow) Upload() {
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		uploadPath := reader.URI().Path()
		mw.ctx.Info(I18n.Sprintf("Selected image file to upload: %s", uploadPath))

		uploadInfo, statErr := os.Stat(uploadPath)
		if statErr == nil && uploadInfo.IsDir() {
			ociLayoutFile := filepath.Join(uploadPath, "oci-layout")
			if _, ociErr := os.Stat(ociLayoutFile); ociErr == nil {
				if layoutErr := mw.ctx.CreateOCILayout(uploadPath); layoutErr != nil {
					dialog.ShowError(fmt.Errorf(I18n.Sprintf("Load OCI layout failed: %v", layoutErr)), mw.window)
					return
				}
				mw.pathUpload = uploadPath
				srcImgUrlList := mw.ctx.OCILayout.ListRefs()
				mw.teInput.SetText(strings.Join(srcImgUrlList, "\n"))
				dialog.ShowInformation(I18n.Sprintf("Start transmit now ?"),
					I18n.Sprintf("Total %v images found in OCI layout, if need update image name or tag, you can cancel and modify the list in the left edit box or we will start upload by default", len(srcImgUrlList)),
					mw.window)
				mw.startUploadOCI()
				return
			}
			dialog.ShowError(fmt.Errorf(I18n.Sprintf("Directory is not a valid OCI layout")), mw.window)
			return
		}

		if IsOCILayoutTar(uploadPath) {
			tmpExtractDir := filepath.Join(TEMP_DIR, "oci-upload-extract")
			oli, extractErr := ExtractFromTar(uploadPath, tmpExtractDir)
			if extractErr != nil {
				dialog.ShowError(fmt.Errorf(I18n.Sprintf("Extract OCI layout tar failed: %v", extractErr)), mw.window)
				return
			}
			mw.ctx.OCILayout = oli
			mw.ctx.StoreFormat = "oci"
			mw.pathUpload = tmpExtractDir
			srcImgUrlList := oli.ListRefs()
			mw.teInput.SetText(strings.Join(srcImgUrlList, "\n"))
			dialog.ShowInformation(I18n.Sprintf("Start transmit now ?"),
				I18n.Sprintf("Total %v images found in OCI layout tar, if need update image name or tag, you can cancel and modify the list in the left edit box or we will start upload by default", len(srcImgUrlList)),
				mw.window)
			mw.startUploadOCI()
			return
		}

		b, readErr := ioutil.ReadFile(uploadPath)
		if readErr != nil {
			dialog.ShowError(fmt.Errorf(I18n.Sprintf("Open file failed: %v", readErr)), mw.window)
			return
		}
		cm := new(CompressionMetadata)
		err = yaml.Unmarshal(b, cm)
		if err != nil {
			dialog.ShowError(fmt.Errorf(I18n.Sprintf("Parse file failed(version incompatible or file corrupt?): %v", err)), mw.window)
			return
		}
		pathname := filepath.Dir(uploadPath)

		for k, v := range cm.Datafiles {
			f, statErr := os.Stat(filepath.Join(pathname, k))
			if statErr != nil && os.IsNotExist(statErr) {
				mw.ctx.Errorf(I18n.Sprintf("Datafile %s missing", filepath.Join(pathname, k)))
				dialog.ShowError(fmt.Errorf(I18n.Sprintf("Some data files missing, please check the log")), mw.window)
				return
			} else if f.Size() != v {
				mw.ctx.Errorf(I18n.Sprintf("Datafile %s mismatch in size, origin: %v, now: %v", filepath.Join(pathname, k), v, f.Size()))
				dialog.ShowError(fmt.Errorf(I18n.Sprintf("Some data files mismatch, please check the log")), mw.window)
				return
			}
		}

		mw.pathUpload = pathname
		mw.cmUpload = cm

		var srcImgUrlList []string
		for k := range cm.Manifests {
			srcImgUrlList = append(srcImgUrlList, k)
		}
		mw.teInput.SetText(strings.Join(srcImgUrlList, "\n"))

		dialog.ShowInformation(I18n.Sprintf("Start transmit now ?"),
			I18n.Sprintf("Total %v images found, if need update image name or tag, you can cancel and modify the list in the left edit box or we will start upload by default", len(cm.Manifests)),
			mw.window)

		mw.startUpload(cm, pathname)
	}, mw.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".yaml", ".tar", ".tgz"}))
	fd.Show()
}

func (mw *MyMainWindow) startUpload(cm *CompressionMetadata, pathname string) {
	go func() {
		mw.BeginAction()
		mw.ctx.CompMeta = cm
		imgList := mw.getInputList()

		if mw.ctx.CompMeta.Compressor == "squashfs" {
			var filename string
			for k := range cm.Datafiles {
				filename = k
			}
			workName := strings.TrimSuffix(filename, ".squashfs")
			if !TestSquashfs() || strings.Contains(CONF.Squashfs, "stream") {
				err := mw.ctx.CreateSquashfsTar(TEMP_DIR, workName, filepath.Join(mw.pathUpload, filename))
				if err != nil {
					fyne.Do(func() {
						dialog.ShowError(fmt.Errorf(I18n.Sprintf("Unsquashfs uncompress failed with %v", err)), mw.window)
					})
					return
				}
			} else {
				mw.ctx.CreateSquashfsTar(TEMP_DIR, workName, "")
				mw.ctx.Info(I18n.Sprintf("Unsquashfs uncompress Start"))
				var err error
				if strings.Contains(CONF.Squashfs, "nocmd") {
					err = UnSquashfs(mw.ctx.GetLogger(), filepath.Join(TEMP_DIR, workName), filepath.Join(mw.pathUpload, filename), true)
				} else {
					err = UnSquashfs(mw.ctx.GetLogger(), filepath.Join(TEMP_DIR, workName), filepath.Join(mw.pathUpload, filename), false)
					mw.ctx.Temp.SavePath(workName)
				}
				mw.ctx.Info(I18n.Sprintf("Unsquashfs uncompress End"))
				if err != nil {
					mw.ctx.Error(I18n.Sprintf("Unsquashfs uncompress failed with %v", err))
					fyne.Do(func() {
						dialog.ShowError(fmt.Errorf(I18n.Sprintf("Unsquashfs uncompress failed with %v", err)), mw.window)
					})
					return
				}
			}
		}

		c, err := NewClient(mw.maxConn, mw.retries, mw.ctx)
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(err, mw.window)
			})
			return
		}

		fyne.Do(func() {
			mw.btnUpload.Disable()
		})
		defer fyne.Do(func() {
			mw.btnUpload.Enable()
		})

		for _, rawURL := range imgList {
			if mw.ctx.Cancel() {
				mw.ctx.Error("User cancelled...")
				return
			}
			src, dst := GenRepoUrl("", mw.dstRepo.Registry, mw.dstRepo.Repository, rawURL)
			c.GenerateOfflineUploadTask(src, dst, mw.pathUpload, mw.dstRepo)
		}

		mw.ctx.UpdateTotalTask(c.TaskLen())
		c.Run()
		mw.EndAction()
	}()
}

func (mw *MyMainWindow) startUploadOCI() {
	go func() {
		mw.BeginAction()
		imgList := mw.getInputList()

		c, err := NewClient(mw.maxConn, mw.retries, mw.ctx)
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(err, mw.window)
			})
			return
		}

		fyne.Do(func() {
			mw.btnUpload.Disable()
		})
		defer fyne.Do(func() {
			mw.btnUpload.Enable()
		})

		for _, rawURL := range imgList {
			if mw.ctx.Cancel() {
				mw.ctx.Error("User cancelled...")
				return
			}
			src, dst := GenRepoUrl("", mw.dstRepo.Registry, mw.dstRepo.Repository, rawURL)
			c.GenerateOfflineUploadTask(src, dst, mw.pathUpload, mw.dstRepo)
		}

		mw.ctx.UpdateTotalTask(c.TaskLen())
		c.Run()
		mw.EndAction()
	}()
}

func (mw *MyMainWindow) Verify() {
	imgList := mw.getInputList()
	if imgList != nil {
		var text = I18n.Sprintf("Image List") + ":\n"
		text = text + I18n.Sprintf("Source Repository") + ", " + I18n.Sprintf("Destination Repository") + "\n"
		for _, rawURL := range imgList {
			src, dst := GenRepoUrl(mw.srcRepo.Registry, mw.dstRepo.Registry, mw.dstRepo.Repository, rawURL)
			text = text + src + ", " + dst + "\n"
		}
		mw.ctx.Info(text)
	}
}

type GuiLogger struct {
	te      *widget.Entry
	logChan chan int
	mu      sync.Mutex
}

func newGuiLogger(te *widget.Entry) CtxLogger {
	return &GuiLogger{
		te:      te,
		logChan: make(chan int, 1),
	}
}

func (l *GuiLogger) append(text string) {
	l.mu.Lock()
	newText := l.te.Text + text
	if len(newText) > 1000000 {
		newText = newText[len(newText)-1000000:]
	}
	l.mu.Unlock()
	fyne.Do(func() {
		l.te.SetText(newText)
	})
}

func (l *GuiLogger) Info(logStr string) {
	l.logChan <- 1
	defer func() { <-l.logChan }()
	l.append(time.Now().Format("[2006-01-02 15:04:05]") + " " + logStr + "\n")
	log.Info(logStr)
}

func (l *GuiLogger) Error(logStr string) {
	l.logChan <- 1
	defer func() { <-l.logChan }()
	l.append(time.Now().Format("[2006-01-02 15:04:05]") + " " + logStr + "\n")
	log.Error(logStr)
}

func (l *GuiLogger) Debug(logStr string) {
	log.Debug(fmt.Sprint(logStr))
}

func (l *GuiLogger) Errorf(format string, args ...interface{}) error {
	l.logChan <- 1
	defer func() { <-l.logChan }()
	var errStr string
	if len(args) > 0 {
		errStr = fmt.Sprintf(format, args...)
	} else {
		errStr = format
	}
	l.append(time.Now().Format("[2006-01-02 15:04:05]<ERROR>") + " " + errStr + "\n")
	log.Error(errStr)
	return fmt.Errorf(errStr)
}
