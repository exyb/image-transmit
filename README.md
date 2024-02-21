# 容器镜像传输工具

## 说明

1. 遵从 Apache 2.0 协议
2. fork 自 wct-devops/image-transmit

## 修改说明

1. 支持 OCI 镜像
2. 配置文件中增加平台和部分自定义选项

## 配置文件

在工具的目录下,放置一个配置文件cfg.yaml(或者放在 data 目录下, 即 `data/cfg.yaml`),其内容参考如下：

```yaml
source: # 源仓库信息配置,可以支持多个
- registry: "http://10.45.80.1"
  user: #用户名和密码，如果匿名访问，用户名和密码都留空即可
  password:
  #name: #可选配置,指定名称
target:  # 目标仓库信息配置,可以支持多个
- registry: "http://10.45.46.109"
  user:
  password:
  #repository: # 可选配置，是否修改镜像名称，假如填写值yyyy，则会将源仓库的10.45.80.1/xxxx/image:tag统一改成10.45.46.109/yyyy/image:tag
  #name: #可选配置,指定名称
platform: linux/amd64 # 保存文件的时候, 只会保存指定平台的镜像
#maxconn: 5 # 可选配置，最大并发数，默认5
#retries: 2 # 可选配置，最大重试次数，默认2
#singlefile: false #可选配置，是否生成单一文件，默认关
#dockerfile: fasle #可选配置，是否保存成Docker兼容的格式，本功能需要同时打开singlefile开关
#compressor: # 可选配置。如果不配置，windows下默认为tar模式, linux下如果系统存在mksquashfs/tar,且运行时为特权账号(root或者sudo)，则采用squashfs模式，否则为tar模式，详细解释参考说明
#lang: en_US # 可选配置，指定语言版本,支持中英文两种语言，默认取操作系统语言
#cache:   # 可选配置，是否开启本地缓存，默认关，详细参考说明
#  pathname: cache # 缓存目录
#  keepdays: 7  # 缓存最长保留时间，默认7
#  keepsize: 10  # 缓存目录最大使用量，单位G，默认10
#outprefix: # 可选配置，用于指定生成的压缩文件的前缀，也可以在执行命令时使用-out参数来指定, 最终前缀为 <outprefix>_<architecture>_<full|incre>_<timestamp>.tar
#interval: 60 # 可选配置，守护模式下定时扫描的时间间隔，默认60
#dingtalk: # 可选配置,用于发送钉钉通知，支持多个
#- token :  # 用于配置钉钉令牌
#  secret:  # 用于配置钉钉密钥
```


## 使用说明

1. 这里主要说明离线模式和增量模式, 其他模式(直传模式, 守护模式)参考源仓库说明
2. 主要说明命令行模式

#### 离线场景

一些客户不允许提供可以开放公网访问的跳板机，因此不能使用直传模式，需要使用离线模式，即先在研发中心将镜像打包成文件，然后转移到跳板机，然后从跳板机上将文件上传到目标仓库。整个过程需要使用到【Save Mode】和【Upload Mode】两个选项

1. 启动镜像下载，程序会在程序所在的目录下按照【日期】创建一个新文件夹，并且根据时间戳创建对应的数据文件和描述文件，示例如下：

```
├─20210612
│      img_full_202106122344_0.tar
│      img_full_202106122344_1.tar
│      img_full_202106122344_meta.yaml
```

用户也可以通过outprefix参数来指定一个自己的前缀。
1. 用户将生成的几个文件全部传输给客户，放置到跳板机的某个目录下。
2. 登录跳板机，选择目标仓库，按需调整并发度和异常重试次数，点击【上传】按钮，在弹出文件框中选择上一步中传输过去的描述文件，确认信息后，开始上传
3. 上传时会根据描述文件，对数据文件进行个数/大小的校验，如果报错，请检查数据文件是否传输完整。

> 关于增量和全量的说明  
> 在离线传输模式下，由于无法同时连接源仓库和目标仓库，因此无法判断需要传输哪些镜像分层，全量模式就是将所有的分层都打包到数据文件中，是最安全的方式，但是这种模式下数据文件会比较大。
> 当选择增量模式时，会自动弹出文件框，要求选择一个【上次传输的描述文件】，会根据这个文件将上次传输过的镜像分层都跳过，只打包没有传输过的分层，因此可以大幅降低包的大小。但是这种模式下在上传增量镜像的时候，需要通过流程保证已经将【上次传输的描述文件】的镜像导入到目标库了，不能跳过。
> 一个建议的增量和全量发布流程如下：
> 1. 每个月初发布一个全量的版本
> 2. 每天发布增量版本，弹出框时选择【月初的全量版本】，这样即可只发增量变更的分层
> 3. 现场保证每次的全量版本必须导入，直接增量则导入需要的增量包即可（不需要将每个增量包逐次导入）
> 4. 在_meta.yaml文件有本次版本依赖的版本的信息，可以进行确认，形如`https://last.img/skip/it:img_full_202106130839_meta.yaml`，
> 4. 开发人员临时发送个别镜像测试时，选择全量模式更安全

> 关于单一文件的说明  
> 通常下载时选择使用几个并发，就会产生几个独立数据文件，但是某些场景下，用户希望只需要一个数据文件，方便传输，则可以打开【单一文件】的开关。打开这个开关后，文件都会先被下载到临时目录中，最后被合并成一个文件，因此占用空间和IO，时间也会更长。
> 如果下载时是在同一个局域网内，不存在网络瓶颈，可以在界面上设置并发度为1或者在cfg.yaml中增加一个maxconn=1的配置，只使用一个网络线程，也可以实现只生成一个文件，并且避免了本地操作，可以验证一下两种方式哪种更快。

## 命令行模式
龙舟版开始同时支持windows和linux下命令行方式进行传输：

1. 由于需要通过命令行来指定源/目标仓库信息，因此建议在配置文件中为每个仓库配置一个name，以方便使用。如果不指定，则默认的name为"registry-repository"
2. 使用命令行时, 可以将镜像列表放到一个文本文件，然后通过-lst参数传递给程序，也可以直接使用输入流的方式来指定，简化日常操作，可以参见下文中的使用示意。

```
zoms@172.16.85.48[/home/zoms]$ image-transmit 
Invalid args, please refer the help
Image Transmit-DragonBoat-WhaleCloud DevOps Team
./image-transmit [OPTIONS]
Examples:
            Save mode:           ./image-transmit -src=nj -lst=img.lst
            Increment save mode: ./image-transmit -src=nj -lst=img.lst -inc=img_full_202106122344_meta.yaml
            Upload mode:           ./image-transmit -dst=gz -img=img_full_202106122344_meta.yaml
More description please refer to github.com/wct-devops/image-transmit
  -dst string
        Destination repository name, default: the first repo in cfg.yaml
  -img string
        Image meta file to upload(*meta.yaml)
  -inc string
        The referred image meta file(*meta.yaml) in increment mode
  -lst string
        Image list file, one image each line
  -src string
        Source repository name, default: the first repo in cfg.yaml
  -out string
        Output filename prefix
zoms@172.16.85.48[/home/zoms]$ image-transmit -src=nj <<EOF
10.45.80.21/public/alpine:3.11
10.45.80.21/public/alpine:3.12.1
10.45.80.21/public/alpine:3.12.2
EOF
[2021-06-14 21:42:12] Get 3 images
[2021-06-14 21:42:12] Create data file: img_full_202106142142_0.tar
[2021-06-14 21:42:12] Create data file: img_full_202106142142_1.tar
[2021-06-14 21:42:14] Generated a download task for 10.45.80.1/public/alpine:3.11
[2021-06-14 21:42:16] Generated a download task for 10.45.80.1/public/alpine:3.12.1
[2021-06-14 21:42:17] Url http://10.45.80.1/public/alpine:3.12.2 format error: Error reading manifest 3.12.2 in 10.45.80.1/public/alpine: manifest unknown: manifest unknown, skipped
[2021-06-14 21:42:17] Start processing taks, total 2 ...
[2021-06-14 21:42:17] Get manifest from 10.45.80.1/public/alpine:3.11
Invalid:1 Total:2 Success:0 Failed:0 Doing:1 Down:0.0B/s Up:0.0B/s, Total Down:0.0B Up:0.0B Time:
[2021-06-14 21:42:17] Get manifest from 10.45.80.1/public/alpine:3.12.1
[2021-06-14 21:42:17] Get a blob sha256:cbdbe7a5bc2a(2.7MB) from 10.45.80.1/public/alpine:3.11 success
[2021-06-14 21:42:17] Get a blob sha256:188c0c94c7c5(2.7MB) from 10.45.80.1/public/alpine:3.12.1 success
[2021-06-14 21:42:17] Get a blob sha256:f70734b6a266(1.5KB) from 10.45.80.1/public/alpine:3.11 success
[2021-06-14 21:42:17] Get a blob sha256:d6e46aa2470d(1.5KB) from 10.45.80.1/public/alpine:3.12.1 success
[2021-06-14 21:42:17] Task completed, total 2 tasks with 0 failed
[2021-06-14 21:42:17] WARNING: there are 1 images failed with invalid url(ex:image not exists)
[2021-06-14 21:42:17] Invalid url list:
http://10.45.80.1/public/alpine:3.12.2
[2021-06-14 21:42:17] Create meta file: data/20210614/img_full_202106142142_meta.yaml
```

> 如何将镜像直接导入到本机的Docker？  
> 通常我们上传的压缩包是直接导入到镜像仓库中，但是有些场景，比如一键部署时，需要将镜像直接导入到本地Docker来使用，这个时候可以直接指定参数-dst=docker就可以将目标设置为本地docker

> 建议使用缓存目录  
> 如果使用一些固定的机器来给项目发布镜像，可以打开缓存，这样可以避免每次重复下载已有的镜像层，大大提高打包的效率

## 版本下载说明
请到 release 页面下载
- image-transmit : Linux命令行版
- image-transmit-cmd.zip : Windows命令行版
- image-transmit-gui.zip : Windows桌面版
- image-transmit-arm.zip : Linux ARM命令行版
- image-transmit-darwin.zip : Darwin命令行版

## 多语言支持
目前支持中英两种语言，会自动根据操作系统的设置自动切换。如果想强制切换，有两种方式：
1. 在cfg.yaml中通过lang参数来指定en_US/zh_CN
2. 在环境变量中指定lang参数

## 致谢
使用到的开源库:
github.com/AliyunContainerService/image-syncer  
github.com/lxn/walk  
github.com/klauspost/compress/zstd  
github.com/ulikunitz/xz  
github.com/pierrec/lz4  
github.com/mcuadros/go-version  
github.com/blinkbean/dingtalk  

碰到问题欢迎大家提issue
