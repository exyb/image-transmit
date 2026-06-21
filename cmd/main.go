package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mcuadros/go-version"
	. "github.com/wct-devops/image-transmit/core"
	"github.com/wct-devops/image-transmit/embedded"
	"gopkg.in/yaml.v2"
)

var (
	end       = false
	srcRepo   *Repo
	dstRepo   *Repo
	imgList   []string
	flConfSrc *string
	flConfDst *string
	flConfLst *string
	flConfInc *string
	flConfImg *string
	flConfFmt *string
	flConfWat *bool
	flOpt     *string
)

func main() {
	if embedded.HandleMulticall() {
		return
	}

	InitI18nPrinter("")

	var loggerCfg []byte
	if _, err := os.Stat("logCfg.xml"); err == nil {
		loggerCfg, _ = ioutil.ReadFile("logCfg.xml")
	} else if _, err := os.Stat(filepath.Join(HOME, "logCfg.xml")); err == nil {
		loggerCfg, _ = ioutil.ReadFile(filepath.Join(HOME, "logCfg.xml"))
	}
	InitLogger(loggerCfg)
	CONF = new(YamlCfg)

	var cfgFile []byte
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)

	// data/cfg.yaml > current folder > executable path
	var cfgPath string
	_, err = os.Stat(filepath.Join(HOME, "cfg.yaml"))
	if err != nil && os.IsNotExist(err) {
		_, err := os.Stat("cfg.yaml")
		if err != nil && os.IsNotExist(err) {
			_, err = os.Stat(filepath.Join(exPath, "cfg.yaml"))
			if err != nil && os.IsNotExist(err) {
				fmt.Println(I18n.Sprintf("Read cfg.yaml failed in following path: executable path, current folder, data/cfg.yaml. %v", err))
			} else {
				cfgPath = filepath.Join(exPath, "cfg.yaml")
			}
		} else {
			cfgPath = "cfg.yaml"
		}
	} else {
		cfgPath = filepath.Join(HOME, "cfg.yaml")
	}
	// cfg.yaml in current folder
	cfgFile, err = ioutil.ReadFile(cfgPath)
	if err != nil {
		fmt.Println(I18n.Sprintf("Read cfg.yaml failed: %v", err))
	}

	if err != nil {
		return
	}

	err = yaml.Unmarshal(cfgFile, CONF)
	if err != nil {
		fmt.Print(I18n.Sprintf("Parse cfg.yaml file failed: %v, for instruction visit github.com/wct-devops/image-transmit", err))
		os.Exit(1)
	}

	if CONF.Platform == "" {
		CONF.Platform = "linux/amd64"
	}

	if CONF.OutPrefix == "" {
		CONF.OutPrefix = "img"
	}

	if CONF.MaxConn == 0 {
		CONF.MaxConn = runtime.NumCPU()
	}

	if CONF.Retries == 0 {
		CONF.Retries = 2
	}

	if CONF.Interval > 0 {
		INTERVAL = CONF.Interval
	}

	if len(CONF.Compressor) == 0 {
		if runtime.GOOS == "windows" {
			CONF.Compressor = "tar"
		} else {
			if TestSquashfs() && TestTar() && (len(os.Getenv("SUDO_UID")) > 0 || os.Geteuid() == 0) {
				CONF.Compressor = "squashfs"
			} else {
				CONF.Compressor = "tar"
			}
		}
	}

	if CONF.OutputFormat == "" {
		CONF.OutputFormat = "both"
	}

	if CONF.LocalStore == "" {
		CONF.LocalStore = filepath.Join(HOME, ".blob-store")
	}

	if CONF.Compressor != "squashfs" {
		SQUASHFS = false
	} else {
		if runtime.GOOS != "windows" {
			if TestSquashfs() && TestTar() && (len(os.Getenv("SUDO_UID")) > 0 || os.Geteuid() == 0) {
				// ok
			} else {
				fmt.Print(I18n.Sprintf("Squashfs condition check failed, we need root privilege(run as root or sudo) and squashfs-tools/tar installed\n"))
				return
			}
		}
	}

	if len(CONF.Lang) > 1 {
		InitI18nPrinter(CONF.Lang)
	}

	flConfSrc = flag.String("src", "", I18n.Sprintf("Source repository name, default: the first repo in cfg.yaml"))
	flConfDst = flag.String("dst", "", I18n.Sprintf("Destination repository name, default: the first repo in cfg.yaml"))
	flConfLst = flag.String("lst", "", I18n.Sprintf("Image list: a file path(one image per line) or comma-separated image names"))
	flConfInc = flag.String("inc", "", I18n.Sprintf("Increment target: OCI layout dir/tar, or comma-separated image names to download"))
	flConfImg = flag.String("img", "", I18n.Sprintf("Image source to load: OCI layout dir, OCI layout tar, or Docker tar"))
	flConfFmt = flag.String("format", "", I18n.Sprintf("Output format: oci, docker, both (default: both)"))
	flConfWat = flag.Bool("watch", false, I18n.Sprintf("Watch mode"))
	flOpt = flag.String("opt", "", I18n.Sprintf("Options: override some options in cfg.yaml"))

	flag.Usage = func() {
		fmt.Println(I18n.Sprintf("\n                           Easy Image Transmit Tool"))
		fmt.Print(I18n.Sprintf("[Usage]: \n"))
		fmt.Print(I18n.Sprintf("            Save mode(OCI+Docker):                  %s -src=nj -lst=img.lst\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Save mode(by image names):              %s -src=nj -lst=nginx:1.0,redis:6.0\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Save mode(OCI only):                    %s -src=nj -lst=img.lst -format=oci\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Save mode(Docker only):                 %s -src=nj -lst=img.lst -format=docker\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Increment(from OCI layout dir):          %s -src=nj -lst=old:tag -inc=/path/to/new-oci-layout\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Increment(from OCI layout tar):          %s -src=nj -lst=old:tag -inc=new_image.tar\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Increment(from image names):             %s -src=nj -lst=nginx:1.0 -inc=nginx:2.0\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Load mode(to docker or ctr):            %s -dst=docker -img=img.tar [-lst=img.lst]\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Load mode(to registry):                 %s -dst=gz -img=img.tar [-lst=img.lst]\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Load mode(from OCI layout):             %s -dst=gz -img=/path/to/oci-layout [-lst=img.lst]\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Transmit mode:                          %s -src=nj -lst=img.lst -dst=gz\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Watch mode:                             %s -src=nj -lst=img.lst -dst=gz --watch\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("[Embedded Tools]:\n"))
		fmt.Print(I18n.Sprintf("            Run as subcommand:   %s skopeo inspect docker://image\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Run via symlink:      ln -s %s skopeo; skopeo inspect docker://image\n", os.Args[0]))
		fmt.Print(I18n.Sprintf("            Container:            skopeo, ctr, crictl, nerdctl, regctl\n"))
		fmt.Print(I18n.Sprintf("            Storage:              mc (MinIO/S3 client)\n"))
		fmt.Print(I18n.Sprintf("            Database:             redis-cli\n"))
		fmt.Print(I18n.Sprintf("[Options]:\n%s\n", os.Args[0]))
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(*flConfSrc) > 0 {
		for _, v := range CONF.SrcRepos {
			if v.Name == *flConfSrc {
				srcRepo = &v
				break
			}
		}
		if srcRepo == nil {
			fmt.Print(I18n.Sprintf("Could not find repo: %s", *flConfSrc))
			return
		}
	}

	CONF.DstRepos = append(CONF.DstRepos, Repo{
		Name: "docker",
	}, Repo{
		Name: "ctr",
	})

	if len(*flConfDst) > 0 {
		for _, v := range CONF.DstRepos {
			if v.Name == *flConfDst {
				dstRepo = &v
				break
			}
		}
		if dstRepo == nil {
			fmt.Print(I18n.Sprintf("Could not find repo: %s", *flConfDst))
			return
		}
	}

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
		lc = NewLocalCache(filepath.Join(HOME, CONF.Cache.Pathname), keepDays, keepSize)
	}

	lt := NewLocalTemp(TEMP_DIR)
	log := NewCmdLogger()

	ctx := NewTaskContext(log, lc, lt)
	ctx.Reset()

	// override most of the config from command line
	opts := strings.Split(*flOpt, ",")
	for _, opt := range opts {
		keyValue := strings.Split(opt, "=")
		if len(keyValue) != 2 {
			continue
		}

		key := keyValue[0]
		value := keyValue[1]
		typeOfYamlCfg := reflect.TypeOf(YamlCfg{})
		immutable := reflect.ValueOf(CONF)
		for i := 0; i < typeOfYamlCfg.NumField(); i++ {
			field := typeOfYamlCfg.Field(i)
			tag := reflect.StructTag(field.Tag)
			yamlTag := tag.Get("yaml")
			// fmt.Printf("field' name is %s, yaml tag is %s\n", field.Name, strings.Split(yamlTag, ","))
			_, found := findStrInSlice(strings.Split(yamlTag, ","), key)
			if found {
				fieldValue := immutable.Elem().FieldByName(field.Name)
				switch field.Type.Kind() {
				case reflect.String:
					fieldValue.SetString(value)
				case reflect.Int:
					fieldValue.SetInt(int64(atoi(value)))
				case reflect.Bool:
					fieldValue.SetBool(value == "true")
				}
			}
		}
	}

	// -format flag overrides -opt and cfg.yaml
	if len(*flConfFmt) > 0 {
		CONF.OutputFormat = strings.ToLower(*flConfFmt)
	}

	if len(CONF.Platform) > 0 {
		ctx.Platform = CONF.Platform
	}

	if len(CONF.DingTalk) > 0 {
		ctx.Notify = NewDingTalkWapper(CONF.DingTalk)
	}

	if len(*flConfSrc) > 0 && len(*flConfDst) > 0 {
		// transmit mode: src -> dst
		err := readImgList(ctx)
		if err != nil {
			os.Exit(1)
		}
		if *flConfWat {
			BeginAction(ctx)
			watch(ctx)
		} else {
			BeginAction(ctx)
			transmit(ctx)
			EndAction(ctx)
		}
	} else if len(*flConfImg) > 0 && len(*flConfDst) > 0 {
		// load mode
		ctx.DockerSaverBlobValidate = false
		err := readImgList(ctx)
		if err != nil {
			os.Exit(1)
		}
		BeginAction(ctx)
		upload(ctx)
		EndAction(ctx)
	} else if len(*flConfSrc) > 0 {
		// save mode, save image listed in image list
		err := readImgList(ctx)
		if err != nil {
			os.Exit(1)
		}
		BeginAction(ctx)
		download(ctx)
		EndAction(ctx)
	} else {
		fmt.Println(I18n.Sprintf("Invalid args, please refer the help"))
		flag.Usage()
	}
}

func readImgList(ctx *TaskContext) error {
	if len(*flConfLst) > 0 {
		_, err := os.Stat(*flConfLst)
		if err == nil {
			b, err := ioutil.ReadFile(*flConfLst)
			if err != nil {
				return ctx.Errorf(I18n.Sprintf("Read image list from file failed: %v", err))
			}
			getInputList(string(b), &imgList)
		} else {
			// read image list string
			for _, imgName := range strings.Split(*flConfLst, ",") {
				imgName = strings.TrimSpace(imgName)
				if imgName == "" {
					continue
				}
				imgList = append(imgList, imgName)
			}
		}
	} else {
		var s string

		done := make(chan bool)
		go func() {
			for {
				var l string
				_, err := fmt.Scanln(&l)
				if len(l) < 1 || err != nil {
					break
				}
				s = s + "\n" + l
			}
			done <- true
		}()

		select {
		case <-done:
			if len(s) > 0 {
				if len(s) > 0 {
					getInputList(s, &imgList)
				}
			}
		case <-time.After(100 * time.Microsecond):
		}

	}

	ctx.Info(I18n.Sprintf("Get %v images", len(imgList)))
	return nil
}

func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func findStrInSlice(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

// parseImageNameTag splits an image reference into (repo, tag).
// Correctly handles registry ports, e.g. "host:5000/repo/img:tag" → ("host:5000/repo/img", "tag").
// If no tag is present, defaults to "latest".
func parseImageNameTag(imageRef string) (repo, tag string) {
	lastColon := strings.LastIndex(imageRef, ":")
	lastSlash := strings.LastIndex(imageRef, "/")
	if lastColon > lastSlash && lastColon >= 0 {
		return imageRef[:lastColon], imageRef[lastColon+1:]
	}
	return imageRef, "latest"
}

func getInputList(inputString string, inputList *[]string) {
	inputString = strings.ReplaceAll(inputString, "\t", "")
	if CheckInvalidChar(strings.ReplaceAll(strings.ReplaceAll(inputString, "\r", ""), "\n", "")) {
		fmt.Println(I18n.Sprintf("Invalid chars in image list"))
		return
	}

	for _, imgName := range strings.Split(strings.ReplaceAll(inputString, "\r", ""), "\n") {
		imgName = strings.TrimSpace(imgName)
		if imgName == "" {
			continue
		}
		*inputList = append(*inputList, imgName)
	}
}

func startReport(ctx *TaskContext) {
	go func() {
		for {
			if end {
				break
			}
			fmt.Println(ctx.GetStatus())
			time.Sleep(1 * time.Second)
		}
	}()
}

func transmit(ctx *TaskContext) error {
	c, err := NewClient(CONF.MaxConn, CONF.Retries, ctx)
	if err != nil {
		ctx.Errorf("%v", err)
		return err
	}
	for _, rawURL := range imgList {
		src, dst := GenRepoUrl(srcRepo.Registry, dstRepo.Registry, dstRepo.Repository, rawURL)
		c.GenerateOnlineTask(src, srcRepo, dst, dstRepo)
	}
	ctx.UpdateTotalTask(c.TaskLen())
	startReport(ctx)
	c.Run()
	end = true
	return nil
}

func watch(ctx *TaskContext) error {
	c, err := NewClient(CONF.MaxConn, CONF.Retries, ctx)
	if err != nil {
		ctx.Errorf("%v", err)
		return err
	}

	if imgList != nil {
		ctx.History, err = NewHistory(HIS_FILE)
		if err != nil {
			ctx.Errorf("%v", err)
			return err
		}
		for {
			for _, rawURL := range imgList {
				if ctx.Cancel() {
					ctx.Errorf(I18n.Sprintf("User cancelled..."))
					break
				}

				src, dst := GenRepoUrl(srcRepo.Registry, dstRepo.Registry, dstRepo.Repository, rawURL)
				srcURL, _ := NewRepoURL(src)
				dstURL, _ := NewRepoURL(dst)

				imageSourceSrc, err := NewImageSource(ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), "", srcRepo.User, srcRepo.Password, !strings.HasPrefix(src, "https") || srcRepo.Insecure)
				if err != nil {
					log.Error(err)
					return err
				}

				tags, err := imageSourceSrc.GetSourceRepoTags()
				if err != nil {
					c.PutAInvalidTask(src)
					ctx.Error(I18n.Sprintf("Fetch tag list failed for %v with error: %v", srcURL, err))
					return err
				}

				for _, tag := range tags {
					newSrcUrl := srcURL.GetRegistry() + "/" + srcURL.GetRepoWithNamespace() + ":" + tag
					newDstUrl := dstURL.GetRegistry() + "/" + dstURL.GetRepoWithNamespace() + ":" + tag
					if version.Compare(tag, srcURL.GetTag(), "<") || ctx.History.Skip(newSrcUrl) {
						continue
					}

					newImgSrc, err := NewImageSource(ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), tag, srcRepo.User, srcRepo.Password, !strings.HasPrefix(src, "https") || srcRepo.Insecure)
					if err != nil {
						c.PutAInvalidTask(newSrcUrl)
						ctx.Error(I18n.Sprintf("Url %s format error: %v, skipped", newSrcUrl, err))
						continue
					}

					newImgDst, err := NewImageDestination(ctx.Context, dstURL.GetRegistry(), dstURL.GetRepoWithNamespace(), tag, dstRepo.User, dstRepo.Password, !strings.HasPrefix(dst, "https") || dstRepo.Insecure)
					if err != nil {
						c.PutAInvalidTask(newSrcUrl)
						ctx.Error(I18n.Sprintf("Url %s format error: %v, skipped", newDstUrl, err))
						continue
					}
					var callback func(bool, string)
					if ctx.Notify != nil {
						callback = func(result bool, content string) {
							if result {
								ctx.Notify.Send(I18n.Sprintf("### Transmit Success\n- Image: %v\n- Stat: %v", newDstUrl, content))
							} else {
								ctx.Notify.Send(I18n.Sprintf("### Transmit Failed\n- Image: %v\n- Error: %v", newDstUrl, content))
							}
						}
					}
					c.PutATask(NewOnlineTaskCallback(newImgSrc, newImgDst, ctx, callback))
					ctx.Info(I18n.Sprintf("Generated a task for %s to %s", newSrcUrl, newDstUrl))
				}
				imageSourceSrc.Close()
			}
			ctx.UpdateTotalTask(ctx.GetTotalTask() + c.TaskLen())
			c.Run()
			fmt.Println(ctx.GetStatus())
			select {
			case <-ctx.Context.Done():
				ctx.Errorf(I18n.Sprintf("User cancelled..."))
				return nil
			case <-time.After(time.Duration(INTERVAL) * time.Second):
				continue
			}
		}
	}
	return nil
}

func download(ctx *TaskContext) error {
	if CONF.MaxConn > len(imgList) {
		CONF.MaxConn = len(imgList)
	}
	c, _ := NewClient(CONF.MaxConn, CONF.Retries, ctx)
	var prefixPathname string
	var prefixFilename string
	if len(CONF.OutPrefix) > 0 {
		prefixPathIdx := strings.LastIndex(CONF.OutPrefix, string(os.PathSeparator))
		if prefixPathIdx > 0 {
			prefixPathname = CONF.OutPrefix[0:prefixPathIdx]
			prefixFilename = CONF.OutPrefix[prefixPathIdx+1:]
		} else {
			prefixFilename = CONF.OutPrefix
		}
	}

	pathname := filepath.Join(HOME, time.Now().Format("20060102"), prefixPathname)
	_, err := os.Stat(pathname)
	if os.IsNotExist(err) {
		os.MkdirAll(pathname, os.ModePerm)
	}

	ctx.CreateCompressionMetadata(CONF.Compressor)

	incImageList := make([]string, 0)
	refImageList := make([]string, 0)

	// modify work name if only one image was got
	workName := time.Now().Format("full_200601021504")

	// Load local blob store for automatic incremental
	if CONF.LocalStore != "" {
		localStorePath := CONF.LocalStore
		if storeInfo, statErr := os.Stat(localStorePath); statErr == nil && storeInfo.IsDir() {
			if localStore, lsErr := NewOCILayout(localStorePath); lsErr == nil {
				for _, blob := range getExistingBlobHexes(localStore) {
					ctx.CompMeta.BlobDone(blob, "local-store")
				}
				ctx.Info(I18n.Sprintf("Loaded %d blobs from local store: %s", len(getExistingBlobHexes(localStore)), localStorePath))
			}
		}
	}

	if len(*flConfInc) > 0 {
		workName = time.Now().Format("incr_200601021504")

		// Parse -inc to get the images to download (new images)
		incInfo, err := os.Stat(*flConfInc)
		if err == nil && incInfo.IsDir() {
			incOCILayout, ociErr := NewOCILayout(*flConfInc)
			if ociErr == nil {
				for _, ref := range incOCILayout.ListRefs() {
					incImageList = append(incImageList, ref)
				}
			}
			if len(incImageList) == 0 {
				return fmt.Errorf(I18n.Sprintf("Directory %s is not a valid OCI layout or contains no images", *flConfInc))
			}
		} else if err == nil {
			b, err := ioutil.ReadFile(*flConfInc)
			if err != nil {
				return fmt.Errorf(I18n.Sprintf("Open file failed: %v", err))
			}
			if IsOCILayoutTar(*flConfInc) {
				tmpExtractDir := filepath.Join(TEMP_DIR, "oci-inc-preload")
				incOCILayout, extractErr := ExtractFromTar(*flConfInc, tmpExtractDir)
				if extractErr == nil {
					for _, ref := range incOCILayout.ListRefs() {
						incImageList = append(incImageList, ref)
					}
				}
			} else {
				getInputList(string(b), &incImageList)
			}
		} else {
			for _, imgName := range strings.Split(*flConfInc, ",") {
				imgName = strings.TrimSpace(imgName)
				if imgName == "" {
					continue
				}
				incImageList = append(incImageList, imgName)
			}
		}

		// Mark -lst images' blobs as done (reference/existing images for layer comparison)
		for _, refImg := range imgList {
			url, _ := GenRepoUrl(srcRepo.Registry, "", "", refImg)
			srcURL, err := NewRepoURL(strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://"))
			if err != nil {
				return ctx.Errorf(I18n.Sprintf("Url %s format error: %v, skipped", url, err))
			}
			is, err := NewImageSource(ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), srcURL.GetTag(),
				srcRepo.User, srcRepo.Password, !strings.HasPrefix(url, "https://") || srcRepo.Insecure)
			if err != nil {
				log.Error(err)
				return err
			}
			srcUrl := fmt.Sprintf("%s/%s:%s", is.GetRegistry(), is.GetRepository(), is.GetTag())
			manifestByte, manifestType, err := is.GetManifest()
			if err != nil {
				return errors.New(I18n.Sprintf("Failed to get manifest from %s error: %v", srcUrl, err))
			}
			ctx.Info(I18n.Sprintf("Get reference manifest from %s", srcUrl))

			blobInfos, _, _, err := is.GetBlobInfos(manifestByte, manifestType)
			if err != nil {
				return errors.New(I18n.Sprintf("Get blob info from %s error: %v", srcUrl, err))
			}
			for _, b := range blobInfos {
				ctx.CompMeta.BlobDone(b.Digest.Hex(), fmt.Sprintf("ref:%s", refImg))
			}
		}

		// Save reference images for workName generation
		refImageList = make([]string, len(imgList))
		copy(refImageList, imgList)

		// Swap: imgList becomes -inc images (to download), -lst was reference only
		imgList = incImageList
	}

	if len(imgList) == 1 {
		prefixFilename = ""
		targetRepo, targetTag := parseImageNameTag(imgList[0])
		targetRepoWithoutNamespace := filepath.Base(targetRepo)
		if len(refImageList) == 1 {
			srcRepo, srcTag := parseImageNameTag(refImageList[0])
			srcRepoWithoutNamespace := filepath.Base(srcRepo)

			if targetRepoWithoutNamespace == srcRepoWithoutNamespace {
				workName = strings.ToLower(strings.ReplaceAll(targetRepoWithoutNamespace, "-", "_")) + "_" + targetTag + "_incr_from_" + srcTag
			} else {
				workName = strings.ToLower(strings.ReplaceAll(targetRepoWithoutNamespace, "-", "_")) + "_" + targetTag + "_incr_from_" + strings.ToLower(strings.ReplaceAll(srcRepoWithoutNamespace, "-", "_")) + srcTag
			}
		} else {
			workName = strings.ToLower(strings.ReplaceAll(targetRepoWithoutNamespace, "-", "_")) + "_" + targetTag + "_full"
		}
	}

	arch := strings.ToLower(strings.Split(CONF.Platform, "/")[1])
	if len(prefixFilename) > 0 {
		workName = prefixFilename + "_" + workName + "_" + arch
	} else {
		workName = workName + "_" + arch
	}

	var ociPath string
	if CONF.OutputFormat == "oci" || CONF.OutputFormat == "docker" || CONF.OutputFormat == "both" {
		ociPath = filepath.Join(pathname, workName)
		if err := ctx.CreateOCILayout(ociPath); err != nil {
			return ctx.Errorf(I18n.Sprintf("Create OCI layout failed: %v", err))
		}
		ctx.Info(I18n.Sprintf("Create OCI layout: %s", ociPath))

		// If -inc is an OCI layout dir/tar, copy its blobs to the new layout
		if len(*flConfInc) > 0 {
			var refOCILayout *OCILayout
			incPath := *flConfInc
			incInfo, err := os.Stat(incPath)
			if err == nil && incInfo.IsDir() {
				refOCILayout, err = NewOCILayout(incPath)
			} else if err == nil && IsOCILayoutTar(incPath) {
				tmpExtractDir := filepath.Join(TEMP_DIR, "oci-inc-extract")
				refOCILayout, err = ExtractFromTar(incPath, tmpExtractDir)
			}
			if refOCILayout != nil {
				if copyErr := ctx.OCILayout.CopyBlobsFrom(refOCILayout); copyErr != nil {
					ctx.Info(I18n.Sprintf("Copy reference blobs failed: %v", copyErr))
				} else {
					for _, blob := range getExistingBlobHexes(refOCILayout) {
						ctx.CompMeta.BlobDone(blob, fmt.Sprintf("ref-oci:%s", incPath))
					}
					ctx.Info(I18n.Sprintf("Copied reference blobs from %s", incPath))
				}
			}
		}
	} else if SQUASHFS {
		ctx.Temp.SavePath(workName)
		ctx.CreateSquashfsTar(TEMP_DIR, workName, "")
	} else {
		workDir := filepath.Join(pathname, filepath.Dir(workName))
		if _, err := os.Stat(workDir); os.IsNotExist(err) {
			os.MkdirAll(workDir, os.ModePerm)
		}
		if CONF.SingleFile {
			ctx.CreateSingleWriter(pathname, workName, "tar")
		} else {
			ctx.CreateTarWriter(pathname, workName, "tar", CONF.MaxConn)
		}
	}
	for _, rawURL := range imgList {
		srcUrl, _ := GenRepoUrl(srcRepo.Registry, "", "", rawURL)
		c.GenerateOfflineDownTask(srcUrl, rawURL, srcRepo)
	}
	startReport(ctx)
	ctx.UpdateTotalTask(c.TaskLen())
	c.Run()
	end = true
	if ctx.OCILayout != nil {
		if err := ctx.OCILayout.SaveIndex(); err != nil {
			ctx.Error(I18n.Sprintf("Save OCI index failed: %v", err))
		}
		ctx.Info(I18n.Sprintf("OCI layout saved to %s", ociPath))

		// Update local blob store with new blobs
		if CONF.LocalStore != "" {
			localStorePath := CONF.LocalStore
			if _, statErr := os.Stat(localStorePath); os.IsNotExist(statErr) {
				os.MkdirAll(localStorePath, os.ModePerm)
			}
			if localStore, lsErr := NewOCILayout(localStorePath); lsErr == nil {
				if copyErr := localStore.CopyBlobsFrom(ctx.OCILayout); copyErr != nil {
					ctx.Info(I18n.Sprintf("Update local store failed: %v", copyErr))
				} else {
					ctx.Info(I18n.Sprintf("Local blob store updated: %s", localStorePath))
				}
			}
		}

		tarName := workName + ".tar"
		tarPath := filepath.Join(pathname, tarName)
		var localStore *OCILayout
		if CONF.LocalStore != "" {
			localStore, _ = NewOCILayout(CONF.LocalStore)
		}
		switch CONF.OutputFormat {
		case "oci":
			ctx.Info(I18n.Sprintf("Packing OCI tar: %s", tarPath))
			if err := ctx.OCILayout.PackToTar(tarPath); err != nil {
				ctx.Error(I18n.Sprintf("Pack OCI tar failed: %v", err))
			} else {
				ctx.Info(I18n.Sprintf("OCI tar created: %s", tarPath))
			}
		case "docker":
			ctx.Info(I18n.Sprintf("Packing Docker tar: %s", tarPath))
			if err := ctx.OCILayout.PackToDockerTar(tarPath, localStore, true); err != nil {
				ctx.Error(I18n.Sprintf("Pack Docker tar failed: %v", err))
			} else {
				ctx.Info(I18n.Sprintf("Docker tar created: %s", tarPath))
			}
		default: // "both" or ""
			ctx.Info(I18n.Sprintf("Packing OCI+Docker tar: %s", tarPath))
			if err := ctx.OCILayout.PackToDockerTar(tarPath, localStore, false); err != nil {
				ctx.Error(I18n.Sprintf("Pack tar failed: %v", err))
			} else {
				ctx.Info(I18n.Sprintf("OCI+Docker tar created: %s", tarPath))
			}
		}
	} else if ctx.SingleWriter != nil {
		ctx.SingleWriter.SetQuit()
		ctx.SingleWriter.Run()
		ctx.SingleWriter.SaveDockerMeta(ctx.CompMeta)
	} else {
		ctx.CloseTarWriter()
	}

	if ctx.OCILayout == nil {
		if ctx.SquashfsTar != nil {
			ctx.Info(I18n.Sprintf("Mksquashfs Compress Start"))
			err := MakeSquashfs(ctx.GetLogger(), filepath.Join(TEMP_DIR, workName), filepath.Join(pathname, workName+".squashfs"))
			ctx.Info(I18n.Sprintf("Mksquashfs Compress End"))
			if err != nil {
				ctx.Error(I18n.Sprintf("Mksquashfs compress failed with %v", err))
				return err
			} else {
				ctx.CompMeta.AddDatafile(workName+".squashfs", 0)
			}
		}
	}
	return nil
}

func upload(ctx *TaskContext) error {
	imgPath := *flConfImg
	imgInfo, err := os.Stat(imgPath)
	if err != nil {
		return ctx.Errorf(I18n.Sprintf("Open path failed: %v", err))
	}

	var pathname string
	var srcImgUrlList []string

	if imgInfo.IsDir() {
		// OCI layout directory
		ociLayoutFile := filepath.Join(imgPath, "oci-layout")
		if _, err := os.Stat(ociLayoutFile); os.IsNotExist(err) {
			return ctx.Errorf(I18n.Sprintf("Directory %s does not appear to be an OCI layout (missing oci-layout file)", imgPath))
		}
		if err := ctx.CreateOCILayout(imgPath); err != nil {
			return ctx.Errorf(I18n.Sprintf("Load OCI layout failed: %v", err))
		}
		pathname = imgPath
		srcImgUrlList = ctx.OCILayout.ListRefs()
		ctx.Info(I18n.Sprintf("The OCI layout contains %v images:\n%s", len(srcImgUrlList), strings.Join(srcImgUrlList, "\n")))
	} else if IsOCILayoutTar(imgPath) {
		tmpExtractDir := filepath.Join(TEMP_DIR, "oci-extract-"+filepath.Base(imgPath))
		oli, err := ExtractFromTar(imgPath, tmpExtractDir)
		if err != nil {
			return ctx.Errorf(I18n.Sprintf("Extract OCI layout tar failed: %v", err))
		}
		ctx.OCILayout = oli
		ctx.StoreFormat = "oci"
		pathname = tmpExtractDir
		srcImgUrlList = oli.ListRefs()
		ctx.Info(I18n.Sprintf("The OCI layout tar contains %v images:\n%s", len(srcImgUrlList), strings.Join(srcImgUrlList, "\n")))
	} else {
		return ctx.Errorf(I18n.Sprintf("File %s is not a valid OCI layout tar. Supported: OCI layout directory, OCI layout tar, or OCI+Docker tar", imgPath))
	}

	if len(imgList) == 0 {
		getInputList(strings.Join(srcImgUrlList, "\n"), &imgList) // if no input list then take the original
	}

	c, _ := NewClient(CONF.MaxConn, CONF.Retries, ctx)
	for _, rawURL := range imgList {
		src, dst := GenRepoUrl("", dstRepo.Registry, dstRepo.Repository, rawURL)
		if dstRepo.Name == "docker" || dstRepo.Name == "ctr" {
			ctx.DockerTarget = dstRepo.Name
			c.GenerateOfflineUploadTask(src, "", pathname, dstRepo)
		} else {
			c.GenerateOfflineUploadTask(src, dst, pathname, dstRepo)
		}
	}
	ctx.UpdateTotalTask(c.TaskLen())
	startReport(ctx)
	c.Run()
	end = true
	return nil
}

func getExistingBlobHexes(oli *OCILayout) []string {
	blobs, _ := oli.ListBlobs()
	var hexes []string
	for _, b := range blobs {
		hexes = append(hexes, b.Hex())
	}
	return hexes
}

func BeginAction(ctx *TaskContext) bool {
	ctx.Info(I18n.Sprintf("==============BEGIN=============="))
	ctx.Info(I18n.Sprintf("Transmit params: max threads: %v, max retries: %v", CONF.MaxConn, CONF.Retries))
	ctx.UpdateSecStart(time.Now().Unix())
	return true
}

func EndAction(ctx *TaskContext) {
	if !CONF.KeepTemp {
		ctx.Temp.Clean()
	}
	ctx.UpdateSecEnd(time.Now().Unix())
	if ctx.Notify != nil {
		ctx.Notify.Send(I18n.Sprintf("### Transmit Task End \n- Stat: %v", ctx.GetStatus()))
	}
	ctx.Info(I18n.Sprintf("===============END==============="))
	log.Flush()
}
