## WINDOWS下的容器镜像传输工具

各团队在给项目发布版本的时候，都会涉及到将容器镜像从公司的官方仓库复制到项目的仓库上，一般我们需要让客户找一台Linux作为跳板机，安装好Docker工具，然后docker pull/docker tag/docker push的方式来进行，这台Linux主机需要有单独申请访问外网的权限。这个申请流程比较麻烦，而更常见的是客户会提供一个Windows跳板机。在Windows上安装Docker比较麻烦，而且由于Docker镜像占用空间较多，还需要定时的清理或者维护跳板机，否则容易出现各种异常。

目前我们版本升级都通过界面实现了一键操作的傻瓜方式，而传输镜像因为涉及网络权限和跳板机，很多还是手工操作。因此开发了这个Windows的镜像传输工具方便大家使用。

工具的优势有：
- 绿色版，无需安装任何三方工具，直接下载一个EXE程序(大小仅5M)到主机，参考样例写一个cfg.yaml配置文件即可运行
- 最小化占用CPU和存储资源，对机器要求比较低，传输效率高
- Windows下的界面操作，不懂Docker命令的小白也可以搞定
- 并发数和重试数支持可配置，可以根据网络条件调整
- 同时支持离线模式和直传模式

## 配置文件
在工具的相同目录下,放置一个配置文件cfg.yaml,其内容参考如下：
```yaml
source: # 源仓库信息配置,可以支持多个
- registry: "http://10.45.80.1"
  user: #用户名和密码，如果匿名访问，用户名和密码都留空即可
  password:
target:  # 目标仓库信息配置,可以支持多个
- registry: "http://10.45.46.109"
  user: 
  password: 
  #repository: # 可选配置，是否修改镜像名称，假如填写值yyyy，则会将源仓库的10.45.80.1/xxxx/image:tag统一改成10.45.46.109/yyyy/image:tag
#maxconn: 5 # 可选配置，最大并发数，默认5
#retries: 2 # 可选配置，最大重试次数，默认2
#singlefile: false #可选配置，是否生成单一文件，默认关
#cache:   # 可选配置，是否开启本地缓存，默认关
#  pathname: cache # 缓存目录
#  keepdays: 7  # 缓存最长保留时间，默认7
#  keepsize: 10  # 缓存目录最大使用量，单位G，默认10
```

## 界面截图
![image](https://user-images.githubusercontent.com/11539396/121794392-4bfd6800-cc3a-11eb-8f5f-b3f87eb49f57.png)

## 使用说明
### 直传场景
假设客户提供的跳板机可以连接公网上的仓库，也可以同时连接内网的仓库，则优先推荐使用这个模式。直传模式逐个从源仓库拉取镜像，然后直接推动到目标仓库，文件不落地，网络带宽有保障即可，同时只会传输目标仓库上不存在的分层，因此效率是最高的，如果直通网络有保障，这个是最简单的模式。
1. 选择源仓库和目标仓库，按需调整并发度和异常重试次数
2. 在左侧输入框输入需要传输的镜像列表，会自动替换镜像列表URL地址信息，统一使用选择的源仓库的URL地址，无需手工去替换
3. 可以点击校验用来验证一下源镜像列表和目标镜像列表信息匹配是否正确
4. 点击【直传】按钮，启动镜像复制，界面会自动刷新日志和实时统计
5. 用户可以点击【停止】中断镜像的传输

> 直传模式下是否启用本地缓存的说明  
> 通常不需要启用本地缓存。启用本地缓存后，每次向目标传输的文件都会在缓存目录中保存一份，下次传输时会优先使用缓存的包。有两种场景使用本地缓存可以有所帮助。
> 1. 每次需要向多个目标仓库同步相同的镜像，使用本地缓存，一些包只需要从源仓库下载一次
> 2. 跳板机网络不稳定，使用缓存可以尽量减少一次重复的包传输

### 离线场景
一些客户不允许提供可以开放公网访问的跳板机，因此不能使用直传模式，需要使用离线模式，即先在研发中心将镜像打包成文件，然后转移到跳板机，然后从跳板机上将文件上传到目标仓库。整个过程需要使用到【下载】和【上传】两个按钮。
1. 选择源仓库，按需调整并发度和异常重试次数，以及增量或者全量模式，是否使用单一文件模式等
2. 在左侧输入框输入需要传输的镜像列表，会自动替换镜像列表URL地址信息，统一使用选择的源仓库的URL地址，无需手工去替换
3. 可以点击校验用来验证一下源镜像列表信息是否正确
4. 点击【下载】按钮，启动镜像下载，程序会在程序所在的目录下按照【日期】创建一个新文件夹，并且根据时间戳创建对应的数据文件和描述文件，示例如下：
```
├─20210612
│      img_full_202106122344_0.tar
│      img_full_202106122344_1.tar
│      img_full_202106122344_meta.yaml
```
5. 用户将生成的几个文件全部传输给客户，放置到跳板机的某个目录下。
6. 登录跳板机，选择目标仓库，按需调整并发度和异常重试次数，点击【上传】按钮，在弹出文件框中选择上一步中传输过去的描述文件，确认信息后，开始上传
7. 上传时会根据描述文件，对数据文件进行个数/大小的校验，如果报错，请检查数据文件是否传输完整。

> 增量和全量的说明  
> 在离线传输模式下，由于无法同时连接源仓库和目标仓库，因此无法判断需要传输哪些镜像分层，全量模式就是将所有的分层都打包到数据文件中，是最安全的方式，但是这种模式下数据文件会比较大。
> 当选择增量模式时，会自动弹出文件框，要求选择一个【上次传输的描述文件】，会根据这个文件将上次传输过的镜像分层都跳过，只打包没有传输过的分层，因此可以大幅降低包的大小。但是这种模式下在上传增量镜像的时候，需要保证已经将【上次传输的描述文件】的镜像导入到目标库了，不能跳过。
> 建议的一个增量和全量发布实践如下：
> 1. 每个月初发布一个全量的版本
> 2. 每天发布增量版本，弹出框时选择【月初的全量版本】，这样即可只发增量变更的分层
> 3. 现场保证每次的全量版本必须导入，增量则根据需要导入即可
> 4. 在_meta.yaml文件有本次版本依赖的版本的信息，可以进行确认，形如`https://last.img/skip/it:img_full_202106130839_meta.yaml`，
> 4. 开发人员临时发送个别镜像测试时，选择全量模式更安全

> 关于单一文件的说明  
> 通常下载时选择使用几个并发，就会产生几个独立数据文件，但是某些场景下，用户希望只需要一个数据文件，方便传输，则可以打开【单一文件】的开关。打开这个开关后，文件都会先被下载到临时目录中，最后被合并成一个文件，因此占用空间和IO，时间也会更长。
> 如果下载时是在同一个局域网内，不存在网络瓶颈，可以在界面上设置并发度为1或者在cfg.yaml中增加一个maxconn=1的配置，只使用一个网络线程，也可以实现只生成一个文件，并且避免了本地操作，可以验证一下两种方式哪种更快。


## 致谢
使用到的开源库:  
https://github.com/AliyunContainerService/image-syncer  
https://github.com/lxn/walk  
https://github.com/klauspost/compress/zstd  
https://github.com/ulikunitz/xz  
https://github.com/pierrec/lz4  
  
碰到问题欢迎大家提issue
