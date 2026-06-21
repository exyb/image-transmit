# 嵌入工具扩展指南

## 架构概览

image-transmit 采用 k3s 风格的 **多调用二进制 (multicall binary)** 模式，将多个运维工具嵌入到单一二进制文件中。

### 两种使用方式

```bash
# 方式1: 子命令模式
image-transmit skopeo inspect docker://alpine:latest
image-transmit mc alias list

# 方式2: 符号链接模式 (与原生工具完全兼容)
ln -s image-transmit skopeo
ln -s image-transmit mc
./skopeo inspect docker://alpine:latest
./mc alias list
```

### 工作原理

```
用户调用 image-transmit [tool] args...
        │
        ▼
  HandleMulticall()
        │
   ┌────┴──────────┐
   │ os.Args[0]    │  符号链接检测: basename 匹配 ToolRegistry
   │ basename匹配?  │  例: ln -s image-transmit skopeo → basename=skopeo
   └────┬──────────┘
        │ 匹配
        ▼
   RunTool(name, args)
        │
        ▼
  GetExtractDir()  ──→  首次运行: 解压 embedded/bin/*.gz → .image-transmit-data/bin/
        │
        ▼
  syscall.Exec()  ──→  进程替换为工具二进制 (Linux)
                     exec.Command()  ──→  子进程 (macOS/Windows)
```

## 当前已嵌入工具

| 工具 | 分类 | 版本 | 说明 |
|------|------|------|------|
| skopeo | container | v1.16.1 | 容器镜像仓库操作 |
| ctr | container | v1.7.28 | containerd CLI |
| crictl | container | v1.31.0 | CRI 运行时 CLI |
| nerdctl | container | v2.0.2 | Docker 兼容的 containerd CLI |
| regctl | container | v0.7.2 | 容器仓库客户端 (regclient) |
| mc | storage | latest | MinIO/S3 对象存储客户端 |
| redis-cli | database | system | Redis 命令行客户端 |

## 如何添加新的嵌入工具

### 第1步: 注册工具

编辑 `embedded/multicall.go`，在 `ToolRegistry` 中添加条目：

```go
var ToolRegistry = []ToolDef{
    // ... 已有工具 ...
    {Name: "usql", Category: "database", Description: "Universal SQL client", Symlink: true},
}
```

字段说明：
- `Name`: 工具名，必须与二进制文件名一致
- `Category`: 分类，用于帮助信息展示
- `Description`: 简短描述
- `Symlink`: 是否允许符号链接模式（通常为 true）

### 第2步: 准备二进制文件

将工具的二进制文件 gzip 压缩后放入 `embedded/bin/` 目录：

```bash
# 工具名不含 '-' (如 skopeo, ctr)
gzip -c skopeo > embedded/bin/skopeo.gz

# 工具名含 '-' (如 redis-cli, my-tool)
# go:embed 不支持文件名含 '-'，需用 '_' 替代，运行时自动解码回 '-'
gzip -c redis-cli > embedded/bin/redis_cli.gz
#                                  ^^^^^^^^^ '_' 替代 '-'
```

**命名规则**: `embedded/bin/{name}.gz`，其中 name 中的 `-` 替换为 `_`

### 第3步: 更新 Makefile

在 `Makefile` 的 `EMBEDDED_TOOLS` 变量中添加工具名：

```makefile
EMBEDDED_TOOLS := skopeo ctr crictl nerdctl regctl mc redis-cli usql
```

### 第4步: 更新下载脚本

在 `scripts/download-tools.sh` 中添加下载逻辑：

```bash
echo "[N/M] Downloading your-tool..."
YOUR_TOOL_URL="https://github.com/org/repo/releases/download/v1.0/tool-linux-amd64"
download "$YOUR_TOOL_URL" "$TMP_DIR/your-tool"
compress_and_place "$TMP_DIR/your-tool" "your-tool"
```

### 第5步: 更新帮助信息

编辑 `cmd/main.go` 中的 `flag.Usage` 函数，在对应分类下添加工具名。

### 第6步: 构建和测试

```bash
# 完整构建 (含嵌入工具)
make build-cli-full

# 测试符号链接模式
cd build && ./your-tool --version

# 测试子命令模式
./image-transmit your-tool --version
```

## 特殊情况: usql 集成指南

usql 是一个通用 SQL 客户端，支持 MySQL、PostgreSQL、SQLite、SQL Server 等数十种数据库。由于它需要从源码编译（带数据库驱动标签），集成过程较为特殊。

### 编译 usql

```bash
# 编译带 most 驱动的 usql (支持 MySQL/PG/SQLite/SQL Server/Oracle 等)
go install -tags 'most' github.com/xo/usql@latest

# 编译完成后二进制位于 $(go env GOPATH)/bin/usql
# 验证
$(go env GOPATH)/bin/usql --version
```

可用的编译标签：

| 标签 | 说明 |
|------|------|
| `most` | 所有稳定驱动 (推荐) |
| `all` | 所有可用驱动 (含实验性) |
| `no_oracle` | 排除 Oracle 驱动 |
| `no_sqlserver` | 排除 SQL Server 驱动 |
| `mysql` | 仅 MySQL 驱动 |
| `postgres` | 仅 PostgreSQL 驱动 |

### 嵌入 usql

```bash
# 1. 编译 usql
go install -tags 'most' github.com/xo/usql@latest

USQL_BIN="$(go env GOPATH)/bin/usql"

# 2. 压缩并放入 embedded/bin/ (usql 不含 '-'，直接使用原名)
gzip -c "$USQL_BIN" > embedded/bin/usql.gz

# 3. 在 embedded/multicall.go 的 ToolRegistry 中添加:
#    {Name: "usql", Category: "database", Description: "Universal SQL client (MySQL/PG/SQLite/...)", Symlink: true}

# 4. 在 Makefile 的 EMBEDDED_TOOLS 中添加 usql

# 5. 在 cmd/main.go 帮助信息中添加

# 6. 重新构建
make build-cli-full
```

### usql 使用示例

```bash
# 连接 MySQL
image-transmit usql mysql://user:pass@host:3306/dbname

# 连接 PostgreSQL
image-transmit usql postgres://user:pass@host:5432/dbname

# 连接 SQLite
image-transmit usql sqlite:///path/to/db.sqlite

# 执行 SQL 命令
image-transmit usql pg://localhost/ -c "SELECT version()"

# 执行 SQL 脚本
image-transmit usql pg://localhost/ -f script.sql

# 符号链接模式
ln -s image-transmit usql
./usql mysql://root@localhost/test
```

## 构建命令参考

```bash
# 普通构建 (不含嵌入工具, ~22MB)
make build-cli

# 完整构建 (含嵌入工具, ~86MB)
make build-cli-full

# 仅下载/更新工具二进制
make download-tools

# 创建符号链接
make symlinks

# 清理构建产物
make clean

# 清理嵌入工具二进制
make clean-tools

# 代码检查
make vet
make fmt
```

## 文件名编码规则

Go 的 `//go:embed` 指令不支持文件名中包含 `-`（连字符），因此采用 `_`（下划线）替代方案：

| 工具名 | 嵌入文件名 | 运行时解码 |
|--------|-----------|-----------|
| `skopeo` | `skopeo.gz` | `skopeo` |
| `redis-cli` | `redis_cli.gz` | `redis-cli` |
| `my-tool` | `my_tool.gz` | `my-tool` |

解码逻辑在 `embedded/embedded_release.go` 的 `decodeName()` 函数中：

```go
func decodeName(name string) string {
    return strings.ReplaceAll(name, "_", "-")
}
```

**注意**: 如果工具名本身包含 `_`（如 `my_tool`），此规则会产生冲突。在实际使用中，大多数运维工具使用 `-` 而非 `_`，因此这不是问题。如果确实需要支持含 `_` 的工具名，需要改用更复杂的编码方案。
