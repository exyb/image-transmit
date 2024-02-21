package core

import (
	"os"
	"runtime"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var I18n *message.Printer
var lang []string

func InitI18nPrinter(defaultLang string) {
	if len(defaultLang) > 0 {
		lang = []string{defaultLang}
	} else {
		if runtime.GOOS != "windows" && len(os.Getenv("LANG")) > 0 {
			lang = append(lang, os.Getenv("LANG"))
		}
	}
	if len(lang) < 1 {
		lang = []string{"en_US"}
	}
	var matcher = language.NewMatcher([]language.Tag{language.English, language.Chinese})
	tag, _ := language.MatchStrings(matcher, strings.Join(lang, ","))
	I18n = message.NewPrinter(tag)
}

func init() {
	message.SetString(language.Chinese, "NO", "否")
	message.SetString(language.Chinese, "Source:", "源仓库:")          // GUI
	message.SetString(language.Chinese, "Source Repository", "源仓库") //LOG
	message.SetString(language.Chinese, "Destination Repository", "目标库")
	message.SetString(language.Chinese, "Image List", "镜像列表")
	message.SetString(language.Chinese, "  SingleFile:", "  单一文件:")
	message.SetString(language.Chinese, "  ArchiveMode:", "  发布模式:")
	message.SetString(language.Chinese, "  MaxThreads:", "  最大并发:")
	message.SetString(language.Chinese, "  LocalCache:", "  本地缓存:")
	message.SetString(language.Chinese, "  Destination:", "  目标仓库:")
	message.SetString(language.Chinese, "  Retries:", "  重试次数:")
	message.SetString(language.Chinese, "==============BEGIN==============", "==============开始==============")
	message.SetString(language.Chinese, "===============END===============", "==============结束==============")
	message.SetString(language.Chinese, "UPLOAD", "上传")
	message.SetString(language.Chinese, "DOWNLOAD", "下载")
	message.SetString(language.Chinese, "Image Transmit-Ghang'e-WhaleCloud DevOps Team", "云雀-镜像传输工具-揽月版-浩鲸DevOps团队")
	message.SetString(language.Chinese, "Transmit params: max threads: %v, max retries: %v", "传输参数: 最大线程数:%v, 错误重试次数:%v")
	message.SetString(language.Chinese, "Save meta yaml file failed: %v", "保存镜像规格文件失败: %v")
	message.SetString(language.Chinese, "FULL", "全量")
	message.SetString(language.Chinese, "OFF", "关")
	message.SetString(language.Chinese, "Invalid char(s) found from the input, please check the text in the left edit box", "列表中含有特殊字符，请仔细检查输入的信息")
	message.SetString(language.Chinese, "CANCEL", "取消")
	message.SetString(language.Chinese, "INCR", "增量")
	message.SetString(language.Chinese, "Status: ", "状态: ")
	message.SetString(language.Chinese, "Selected the history image meta file: %s", "选择历史镜像规格文件为: %s")
	message.SetString(language.Chinese, "Selected image meta file to upload: %s", "选择上传的镜像规格文件为: %s")
	message.SetString(language.Chinese, "ON", "开")
	message.SetString(language.Chinese, "The image list is empty, please input on the left edit box, one image each line", "待传输列表为空，请在左边输入框输入镜像列表，一行为一条")
	message.SetString(language.Chinese, "Some data files missing, please check the log", "数据文件不全，请检查是否传输异常")
	message.SetString(language.Chinese, "Datafile %s mismatch in size, origin: %v, now: %v", "数据文件%s大小不一致, 原大小:%v, 现大小:%v")
	message.SetString(language.Chinese, "Datafile %s missing", "数据文件: %s 缺失")
	message.SetString(language.Chinese, "Open file failed: %v", "文件打开错误: %v")
	message.SetString(language.Chinese, "Image List(seperated by lines): ", "请输入传输列表(多行分割): ")
	message.SetString(language.Chinese, "Log Output: ", "日志信息输出: ")
	message.SetString(language.Chinese, "YES", "是")
	message.SetString(language.Chinese, "Start transmit now ?", "是否立即上传?")
	message.SetString(language.Chinese, "Failed to set 'MaxThreads' with error: %v", "最大并发数填写错误，请检查，错误信息: %v")
	message.SetString(language.Chinese, "Total %v images found, if need update image name or tag, you can cancel and modify the list in the left edit box or we will start upload by default", "本压缩包包含%v个镜像, 如果需要修改镜像名和Tag，请取消，并在左侧录入框修改，默认会继续上传")
	message.SetString(language.Chinese, "Continue upload(Yes) or create a new upload(No) or Cancel ?", "继续上传当前镜像(是) 或者 新建一个上传(否) 或者 取消操作？")
	message.SetString(language.Chinese, "Continue the upload ?", "继续上传？")
	message.SetString(language.Chinese, "VERIFY", "校验")
	message.SetString(language.Chinese, "Create meta file: %s", "生成压缩规格文件: %s")
	message.SetString(language.Chinese, "User cancel it", "用户发起取消")
	message.SetString(language.Chinese, "User cancelled...", "用户取消操作")
	message.SetString(language.Chinese, "TRANSMIT", "直传")
	message.SetString(language.Chinese, "Stat data file failed: %v", "统计数据文件信息失败: %v")
	message.SetString(language.Chinese, "Meta file error", "镜像规格文件错误")
	message.SetString(language.Chinese, "Parse file failed(version incompatible or file corrupt?): %v", "解析规格文件失败(可能为版本不同或者文件受损，请检查核对)： %v")
	message.SetString(language.Chinese, "Parse cfg.yaml file failed: %v, for instruction visit github.com/wct-devops/image-transmit", "解析cfg.yaml配置文件失败，使用说明可以咨询DevOps团队或者查看github.com/wct-devops/image-transmit")
	message.SetString(language.Chinese, "Configuration File Error", "配置文件错误")
	message.SetString(language.Chinese, "Configuration File cfg.yaml incorrect, for instruction visit github.com/wct-devops/image-transmit", "配置文件错误, 使用说明可以咨询DevOps团队或者查看github.com/wct-devops/image-transmit")
	message.SetString(language.Chinese, "Please select a history image meta file for increment", "请选择一个历史镜像规格文件")
	message.SetString(language.Chinese, "Read cfg.yaml failed: %v", "读取cfg.yaml报错: %v")
	message.SetString(language.Chinese, "Verify input failed", "输入校验失败")
	message.SetString(language.Chinese, "Input Error", "输入错误")
	message.SetString(language.Chinese, "Choose File Failed: %v", "选择文件发生错误: %v")
	message.SetString(language.Chinese, "Failed to set 'Retries' with error: %v", "重试次数填写错误，请检查，错误信息: %v")
	message.SetString(language.Chinese, "ERROR", "错误")
	message.SetString(language.Chinese, "Please choose an image meta file to upload", "请选择需要上传的镜像规格文件")
	message.SetString(language.Chinese, "Image meta file (*meta.yaml)|*meta.yaml|all (*.*)|*.*", "镜像规格文件 (*meta.yaml)|*meta.yaml|所有文件 (*.*)|*.*")
	message.SetString(language.Chinese, "Skip blob: %s", "跳过blob: %s")
	message.SetString(language.Chinese, "Get a blob %s(%v) from %s success", "下载blob %s(%v) 自 %s 完成")
	message.SetString(language.Chinese, "Save Stream file to cache failed: %v", "保存文件到缓存目录失败: %v")
	message.SetString(language.Chinese, "Reuse cache: %s", "复用缓存 %s")
	message.SetString(language.Chinese, "Put file to archive: %s", "记入待归档列表: %s")
	message.SetString(language.Chinese, "Save Stream file to temp failed: %v", "保存文件到临时目录失败: %v")
	message.SetString(language.Chinese, "Read file from cache failed: %v", "读取缓存文件失败: %v")
	//message.SetString(language.Chinese, "Unknown media type: %s",   "不支持的MediaType类型: %s")
	message.SetString(language.Chinese, "Manifest format error: %v, manifest: %s", "Manifest格式错误: %v, manifest: %s")
	message.SetString(language.Chinese, "Check blob %s(%v) to %s exist error: %v", "检查blob %s(%v)于 %s 是否存在时发生错误: %v ")
	message.SetString(language.Chinese, "Blob %s(%v) has been pushed to %s, will not be pulled", "blob %s(%v) 已经存在于 %s,跳过")
	message.SetString(language.Chinese, "Blob %s size mismatch, size in meta: %v, size in tar: %v", "Blob %s 大小不一致, 记录大小: %v, tar包中实际大小: %v")
	message.SetString(language.Chinese, "Put blob %s(%v) to %s success", "上传blob %s(%v)到 %s 完成")
	message.SetString(language.Chinese, "Put blob %s(%v) to %s failed: %v", "上传blob %s(%v)到 %s 时报错:%v")
	message.SetString(language.Chinese, "Blob not found in datafiles: %s", "在数据文件中找不到Blob: %s")
	message.SetString(language.Chinese, "Put manifest to %s error: %v", "上传manifest到 %s 时报错: %v")
	message.SetString(language.Chinese, "Put manifest to %s", "上传manifest到 %s 完成")
	message.SetString(language.Chinese, "Failed to get manifest from %s error: %v", "从 %s 获取manifest信息时报错: %v")
	message.SetString(language.Chinese, "Get manifest from %s", "从 %s 下载manifest完成")
	message.SetString(language.Chinese, "Get blob info from %s error: %v", "获取 %s Blob信息报错: %v")
	message.SetString(language.Chinese, "Get blob %s(%v) from %s failed: %v", "下载blob %s(%v)于%s时报错: %v ")
	message.SetString(language.Chinese, "Get manifest %v of OS:%s Architecture:%s for manifest list error: %v", "获取 OS:%s Architecture:%s 的manifest %v 时报错: %v")
	message.SetString(language.Chinese, "Put manifestList to %s error: %v", "上传manifestList到 %s 时报错: %v")
	message.SetString(language.Chinese, "Put manifestList to %s", "保存manifestList到 %s 完成")
	message.SetString(language.Chinese, "Transmit successfully from %s to %s", "传输 %s 到 %s 完成")
	message.SetString(language.Chinese, "Start processing taks, total %v ...", "开始处理任务，总计%v个，请稍后...")
	message.SetString(language.Chinese, "Start gzRetries failed tasks", "开始重试异常任务，请稍后 ...")
	message.SetString(language.Chinese, "Task completed, total %v tasks with %v failed", "任务处理结束，总计%v任务，%v任务失败")
	message.SetString(language.Chinese, "Failed tasks:\r\n%s", "失败列表:\r\n%s")
	message.SetString(language.Chinese, "Url %s format error: %v, skipped", "URL %s 格式错误: %v, 传输跳过")
	message.SetString(language.Chinese, "Generated a task for %s to %s", "完成从 %s 到 %s 的任务创建")
	message.SetString(language.Chinese, "Generated a download task for %s", "完成 %s下载任务创建")
	message.SetString(language.Chinese, "Generated a upload task for %s", "完成 %s 上传任务创建")
	message.SetString(language.Chinese, "Create data file: %s", "生成数据文件: %s")
	message.SetString(language.Chinese, "All:%v Invalid:%v OK:%v Err:%v Doing:%v Speed:v%s/s ^%s/s Total:v%s ^%s Time:%s", "总计:%v 无效:%v 成功:%v 失败:%v 处理中:%v 速度:下%s/s 上%s/s 总计:下%s 上%s 耗时:%s")
	message.SetString(language.Chinese, "Day", "天")
	message.SetString(language.Chinese, "Hou", "时")
	message.SetString(language.Chinese, "Min", "分")
	message.SetString(language.Chinese, "Sec", "秒")
	message.SetString(language.Chinese, "Source repository name, default: the first repo in cfg.yaml", "源仓库名称, 默认为配置文件中的第一个仓库")
	message.SetString(language.Chinese, "Destination repository name, default: the first repo in cfg.yaml", "目标仓库名称, 默认为第一个")
	message.SetString(language.Chinese, "Image list file, one image each line", "镜像列表文件,一行一个")
	message.SetString(language.Chinese, "The referred image meta file(*meta.yaml) in increment mode", "指定增量模式下参考的镜像规格文件(*meta.yaml)")
	message.SetString(language.Chinese, "Image meta file to upload(*meta.yaml)", "需要上传的镜像规格文件(*meta.yaml)")
	message.SetString(language.Chinese, "%s [OPTIONS]\n", "%s [选项]\n")
	message.SetString(language.Chinese, "Examples: \n", "例子: \n")
	message.SetString(language.Chinese, "            Save mode:           %s -src=nj -lst=img.lst\n", "            下载模式:           %s -src=nj -lst=img.lst\n")
	message.SetString(language.Chinese, "            Increment save mode: %s -src=nj -lst=img.lst -inc=img_full_202106122344_meta.yaml\n", "            增量下载模式: 	%s -src=nj -lst=img.lst -inc=img_full_202106122344_meta.yaml\n")
	message.SetString(language.Chinese, "            Transmit mode:       %s -src=nj -lst=img.lst -dst=gz\n", "            直传模式:           %s -src=nj -lst=img.lst -dst=gz\n")
	message.SetString(language.Chinese, "            Watch mode:          %s -src=nj -lst=img.lst -dst=gz --watch\n", "            守护模式:           %s -src=nj -lst=img.lst -dst=gz --watch\n")
	message.SetString(language.Chinese, "            Upload mode:         %s -dst=gz -img=img_full_202106122344_meta.yaml [-lst=img.lst]\n", "            上传模式:           %s -dst=gz -img=img_full_202106122344_meta.yaml [-lst=img.lst]\n")
	message.SetString(language.Chinese, "More description please refer to github.com/wct-devops/image-transmit\n", "更多帮助可以访问github.com/wct-devops/image-transmit\n")
	message.SetString(language.Chinese, "Could not find repo: %s", "无法找到仓库: %s")
	message.SetString(language.Chinese, "Invalid args, please refer the help", "命令行参数不正确，请查看帮助")
	message.SetString(language.Chinese, "Read image list from file failed: %v", "从文件读取镜像列表失败: %v")
	message.SetString(language.Chinese, "Read image list from stdin failed: %v", "从stdin读取镜像列表失败: %v")
	message.SetString(language.Chinese, "Empty image list", "镜像列表为空")
	message.SetString(language.Chinese, "Get %v images", "读取%v个镜像")
	message.SetString(language.Chinese, "Invalid chars in image list", "镜像列表中含有非法字符,请检查")
	message.SetString(language.Chinese, "The img file contains %v images:\n%s", "镜像文件包含 %v 个镜像:\n%s")
	message.SetString(language.Chinese, "Invalid url list:\r\n%s", "无效镜像列表:\r\n%s")
	message.SetString(language.Chinese, "WARNING: there are %v images failed with invalid url(Ex:images not exist)", "警告: 有%v个镜像因为URL非法而传输失败(如镜像不存在)")
	message.SetString(language.Chinese, "Task failed with %v", "任务执行失败: %v")
	message.SetString(language.Chinese, "Mksquashfs compress failed with %v", "Mksquashfs压缩失败: %v")
	message.SetString(language.Chinese, "Unsquashfs uncompress failed with %v", "Unsquashfs解压失败: %v")
	message.SetString(language.Chinese, "Mksquashfs Compress Start", "Squashfs压缩开始")
	message.SetString(language.Chinese, "Mksquashfs compress End", "Squashfs压缩结束")
	message.SetString(language.Chinese, "Unsquashfs uncompress Start", "Squashfs解压开始")
	message.SetString(language.Chinese, "Unsquashfs uncompress End", "Squashfs解压结束")
	message.SetString(language.Chinese, "Squashfs condition check failed, we need root privilege(run as root or sudo) and squashfs-tools/tar installed\n", "Squashfs条件检查失败，当使用squashfs压缩时需要使用sudo或者root账号运行，并且安装好squashfs-tools和tar工具\n")
	message.SetString(language.Chinese, "Output filename prefix", "输出压缩文件的前缀")
	message.SetString(language.Chinese, "WATCH", "守护")
	message.SetString(language.Chinese, "Fetch tag list failed for %v with error: %v", "获取%v的tag列表失败: %v")
	message.SetString(language.Chinese, "Speed:^%s/s v%s/s Total:^%s v%s", "速度:上%s/s 下%s/s 传输总量:上%s 下%s")
	message.SetString(language.Chinese, "### Transmit Task End \n- Stat: %v", "### 镜像传输结束 \n- 统计: %v")
	message.SetString(language.Chinese, "### Transmit Success\n- Image: %v\n- Stat: %v", "### 镜像传输完成\n- 镜像: %v\n- 统计: %v")
	message.SetString(language.Chinese, "### Transmit Failed\n- Image: %v\n- Error: %v", "### 镜像传输失败\n- 镜像: %v\n- 错误: %v")
	message.SetString(language.Chinese, "Image Transmit Notify", "镜像传输结果通知")
	message.SetString(language.Chinese, "Send DingTalk message, text: %v", "发送钉钉通知: %v")
	message.SetString(language.Chinese, "Watch mode", "守护模式")
	message.SetString(language.Chinese, "Start Retry failed tasks", "开始重试失败任务")
}
