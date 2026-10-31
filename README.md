# ska-blog-go — SSH 博客

Go + [gliderlabs/ssh](https://github.com/gliderlabs/ssh) + [bubbletea](https://github.com/charmbracelet/bubbletea) 实现的 SSH 博客。默认监听 `127.0.0.1:2222`，访客可用**任意用户名免密**直接进入（guest 模式默认开启）。样式来自 [lipgloss](https://github.com/charmbracelet/lipgloss)，共 3 套内置配色。

---

## 访客使用

### 连接

```bash
ssh -p 2222 <主机>
```

- 端口默认 `2222`（可用 `-addr` 改，见维护者一节）。
- **任意用户名均可**：`ssh zhangsan@主机 -p 2222`、`ssh -l guest 主机 -p 2222` 都能进。
- 连接后进入三栏 TUI：左主列（文章列表/正文）＋右侧 30% Sidebar（STATUS 框＋"提示"键位面板），根边框左上角标题 `SKA-BLOG://GO`、底部为纯边框（无键位文字——键位提示已全部搬进侧栏"提示"面板，见 `internal/tui/layout.go` `bannerBottom`）。
- 退出：`q` 或 `Ctrl+C`。

### 键位表（列表视图）

| 键 | 功能 | 来源（case 分支） |
|---|---|---|
| `↑` / `k` | 选中上移一张卡片 | `internal/tui/app.go` `updateListKey` `case "up", "k"` |
| `↓` / `j` | 选中下移一张卡片 | `internal/tui/app.go` `updateListKey` `case "down", "j"` |
| `Enter` | 进入当前文章阅读态 | `internal/tui/app.go` `updateListKey` `case "enter"` |
| `/` | 进入搜索态（输入关键词实时过滤） | `internal/tui/app.go` `updateListKey` `case "/"` |
| `:` | 进入编号跳转态（输入文章 ID） | `internal/tui/app.go` `updateListKey` `case ":"` |
| `Tab` | 收起/展开右侧侧栏 | `internal/tui/app.go` `Update` `case "tab"` |
| `Ctrl+T` | 循环主题（默认→aura→冷青→默认） | `internal/tui/app.go` `Update` `case "ctrl+t"` |
| `q` | 退出 | `internal/tui/app.go` `updateListKey` `case "q"` |
| `Ctrl+C` | 退出 | `internal/tui/app.go` `Update` `case "ctrl+c"` |

列表滚动跟随：选中卡片始终整块可见，滑出窗口顶/底时窗口整体平移。

### 键位表（搜索态 `/`）

| 键 | 功能 | 来源（case 分支） |
|---|---|---|
| 字符 | 输入关键词（实时过滤标题/正文/ID） | `internal/tui/app.go` `updateInput` 普通字符 rune 累积分支 |
| `Backspace` / `Ctrl+H` | 删除末字符（焦点回首个命中） | `internal/tui/app.go` `updateInput` `case "backspace", "ctrl+h"` |
| `↑` / `↓` | 在命中项间移动光标 | `internal/tui/app.go` `updateInput` `case "up"` / `case "down"` |
| `Enter` | 打开光标所在命中项 | `internal/tui/app.go` `updateInput` `case "enter"` |
| `Esc` | 取消搜索，恢复全量列表 | `internal/tui/app.go` `updateInput` `case "esc"` |

注意：搜索态下 `j`/`k` 是查询字符（搜 `jdk`/`kotlin` 这类词不被截获），光标移动只用方向键。

### 键位表（跳转态 `:`）

| 键 | 功能 | 来源（case 分支） |
|---|---|---|
| 字符 | 输入文章 ID（先精确后前缀匹配） | `internal/tui/app.go` `updateInput` 普通字符 rune 累积分支 |
| `Backspace` / `Ctrl+H` | 删除末字符 | `internal/tui/app.go` `updateInput` `case "backspace", "ctrl+h"` |
| `Enter` | 命中直达阅读；未命中显示"无此文"并保留输入 | `internal/tui/app.go` `updateInput` `case "enter"` |
| `Esc` | 取消跳转 | `internal/tui/app.go` `updateInput` `case "esc"` |

跳转态下方向键不移动（列表未过滤，移动无意义）；`j`/`k` 仍是普通字符（`:wj2` 这类含 j/k 的 ID 可正常输入）。

### 键位表（阅读视图）

| 键 | 功能 | 来源（case 分支） |
|---|---|---|
| `↑` / `k` | 向上滚一行 | `internal/tui/reader.go` `keys` `case "up", "k"` |
| `↓` / `j` | 向下滚一行 | `internal/tui/reader.go` `keys` `case "down", "j"` |
| `←` / `h` / `Esc` | 返回列表视图 | `internal/tui/app.go` `Update` `case "left", "h", "esc"` |
| `Tab` | 收起/展开右侧侧栏 | `internal/tui/app.go` `Update` `case "tab"` |
| `Ctrl+T` | 循环主题 | `internal/tui/app.go` `Update` `case "ctrl+t"` |
| `q` | 退出 | `internal/tui/reader.go` `keys` `case "q"` |
| `Ctrl+C` | 退出 | `internal/tui/app.go` `Update` `case "ctrl+c"` |

阅读态顶部有进度行（`标题 (N%)`）；`PgUp`/`PgDn`/`Space`/`Home`/`End`/`g`/`G` 一律不响应（滚动只留方向键，见 `internal/tui/reader.go` `keys` 注释）。键位提示在右侧 Sidebar 的"提示"面板（`↑↓/j/k 滚动  ←/h/Esc 返回列表  Tab 侧栏  Ctrl+T 主题  Q/Ctrl+C 退出`），根框底部 `bannerBottom` 为纯边框 `└─…─┘`、无键位文字。

### 非交互纯文本列表（无 PTY）

不带 `-tt` 请求时（无 PTY），服务端不进入 TUI，直接输出纯文本文章索引后退出：

```bash
ssh -T -p 2222 <主机>
```

输出格式（每行一篇，`-list` 调试模式同款，见 `internal/server/server.go` `PlainList`）：

```
[wj1] hello,world  (1 段)
[wj2] github clone 加速  (14 段)
[a01] 免费域名  (2 段)
```

`-T` 强制禁用伪终端，`-t` 强制分配伪终端（进 TUI 必须带 PTY）。

---

## 维护者

### 构建与运行

```bash
go build ./...          # 编译全部包
go vet ./...            # 静态检查
go test ./...           # 全部测试（content/render/server/theme/tui）
go run .                # 启动 SSH 服务器（默认 127.0.0.1:2222）
```

用 Makefile 亦可：`make build`、`make test`、`make vet`、`make run`、`make tidy`、`make deploy-arm64`。

#### 命令行 flag（`main.go` 核对）

| flag | 默认值 | 说明 |
|---|---|---|
| `-addr` | `127.0.0.1:2222` | SSH 监听地址 |
| `-articles` | `articles/articles.md` | 文章 Markdown 数据文件路径 |
| `-guest` | `true` | 允许匿名访客（`false` 时启动即报错：guest 鉴权暂未实现） |
| `-list` | `false` | 调试：打印纯文本文章索引（格式同 `ssh -T` 输出）后退出 |
| `-render` | `""` | 调试：渲染指定 ID 的文章 ANSI 文本后退出，如 `go run . -render wj2` |
| `-local` | `false` | 调试：在本终端运行 TUI（不走 SSH，需真 TTY；无 TTY 报 `could not open a new TTY`） |

示例：

```bash
go run . -addr 0.0.0.0:2222               # 绑定公网/容器 2222
go run . -articles my/other.md            # 换数据文件
go run . -guest=false                     # 报错：非 guest 模式暂未实现鉴权
go run . -list                            # 打印文章索引
go run . -render wj2                      # 渲染 wj2 的 ANSI 正文
go run . -local                           # 本地真终端跑 TUI
```

### 文章编辑（`articles/articles.md`）

格式规则（实现见 `internal/content/parse.go`，规则说明在 `SPEC.md` 文章源一节）：

- `# ` 一级标题是博客总标题，**跳过**（不产生文章）。
- 每个 `## ` 开启**一篇文章**；标题行按**第一个空格或 Tab** 切分 ID 与标题，例如 `## wj2  github clone 加速`（多空格）与 `## wj1\thello,world`（Tab 脏格式）都能得到 `ID=wj2 / ID=wj1`。
  - 若标题开头片段不是 ID 形态（仅由 ASCII 字母/数字/`-`/`_` 组成），则整行视为标题，ID 自动生成 `a01`、`a02`…（如 `## 免费域名` → ID=`a01`）。
- `### ` 三级标题及其后所有内容都归入**当前文章的正文**（Body），直到下一个 `## `。
- 正文里 `###`、代码块、列表等原样交给渲染层；没有 `## ` 开头的文章时解析报 `未找到任何文章`。

**换文章**：直接编辑 `articles/articles.md`，重启服务即可；`-articles` 可指向别的文件。列表卡片显示标题＋正文首段摘要＋ID 标签，阅读时长按 `ceil(字数/400)` 估算。

### 部署到板子（arm64）

> **用 Makefile 自动交叉编译 + 打包**：`make deploy-arm64` 一步产出 `/tmp/sshblog-deploy/sshblog-arm64.tar.gz`（内含 linux/arm64 静态二进制 `sshblog`，命名与下方 Dockerfile 的 `COPY sshblog` 一致），随后仍需自行 `scp` 上传到板子。手工操作步骤如下。

1. **交叉编译**（本机，amd64 上产出 linux/arm64 静态二进制，命名 `sshblog` 与 Dockerfile 一致）：

   ```bash
   GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o sshblog .
   ```

2. **打包 + 上传**（可选 tar 压缩）：

   ```bash
   tar -czf sshblog-arm64.tar.gz sshblog
   scp sshblog-arm64.tar.gz user@板子IP:~/
   ```

3. **板子上跑 docker**（板子需已装 docker 与 Dockerfile）。容器内工作目录 `/app`，host key 落在 `/app/data/host_key`：

   ```dockerfile
   FROM scratch
   COPY sshblog /app/sshblog
   WORKDIR /app
   VOLUME /app/data
   EXPOSE 2222
   ENTRYPOINT ["/app/sshblog"]
   ```

   ```bash
   docker build -t skablog .
   docker run -d --name skablog -p 2222:2222 \
     -v ~/sshblog-data:/app/data \
     --restart unless-stopped skablog
   ```

   - 卷挂载 `~/sshblog-data:/app/data`：持久化 host key，容器重建不丢指纹。
   - `--restart unless-stopped`：开机自启，崩溃自动拉起。
   - `-p 2222:2222` 映射 SSH 端口；需保证容器内 `-addr` 监听 `0.0.0.0:2222`（如 `docker run ... skablog -addr 0.0.0.0:2222`，ENTRYPOINT 后可带参数）。

#### host key 迁移注意

`data/host_key`（相对工作目录）由服务端首次启动自动生成，用于识别本机身份。**迁移/重建容器时把旧 host key 一并带上**：

- 复制旧机器的 `data/host_key` 到新机器的 `~/sshblog-data/host_key`（对应容器内 `/app/data/host_key`）。
- 若不迁移，客户端首次连接新 host key 会报 `REMOTE HOST IDENTIFICATION HAS CHANGED` / 指纹变化提示，需 `ssh-keygen -R [主机]:2222` 清除旧记录后重连。

### 常见问题（FAQ）

- **终端全黑白、切主题看不出变化**：多为客户端 `TERM` 与服务端探测不兼容（服务端进程无 `TERM` 时按 stdout 探测，守护进程/docker 下可能退化为无色档）。此时看 STATUS 框的 **`主题：<名字>`** 一行——它是纯文字兜底，无论是否有色都会随 `Ctrl+T` 变化，可用它确认主题切换生效。若需恢复颜色，让客户端以 `TERM=xterm-256color` 或 `COLORTERM=truecolor` 连接（如 `tmux` 会话内 `ssh -tt`），或在板子上用 systemd/tmux 赋予服务端一个真实 TTY 与 `TERM`。
- **白屏 / 一直 `loading...`**：`View()` 在收到 `tea.WindowSizeMsg` 前返回 `"loading..."` 占位，绝不返回空串。排查入口：
  1. 确认带 PTY 连接（`ssh -tt -p 2222 主机`，非 `-T`）。
  2. 确认终端尺寸可上报（tmux/screen 正常；纯管道/无 PTY 会走纯文本列表分支）。
  3. 看服务端进程 stdout 是否有异常（`-local` 无 TTY 会报 `could not open a new TTY`）。
- **`ssh -T` 与 TUI 行为不同**：`-T` 无 PTY → 纯文本列表；进 TUI 必须带 PTY（`-tt`）。
- **鼠标滚轮为什么没反应**：服务端未启用鼠标接管（`internal/server/server.go` 不传 `WithMouseCellMotion`，raw 流不发 `?1000`/`?1006` 上报），终端滚轮不会被 TUI 捕获——这是刻意为之：保住终端自身的拖拽选中即可复制文本。列表/阅读滚动只用 `↑↓`/`j/k`。
- **搜索态怎么选命中项**：`/` 进入搜索后输入关键词实时过滤，`↑`/`↓` 在命中项间移动光标，`Enter` 打开光标所在项；`j`/`k` 是查询字符（搜 `jdk`/`kotlin` 这类含 j/k 的词不会被截获为移动）。
- **输入中文查询为什么依赖客户端**：查询词按 rune 级累积（`internal/tui/app.go` `updateInput`），客户端需把中文按 UTF-8 字节送出（SSH/终端默认即 UTF-8），服务端按 rune 解码后逐字过滤标题/正文/ID。

---

## 项目结构

与 `SPEC.md` 的文件职责表核对一致：

```
ska-blog-go/
├── main.go              # flag: -addr/-articles/-guest(默认true)/-list/-render/-local; 模式分流(默认server, -local调试)
├── internal/content/    # articles.md 解析层(含测试): # 跳过/## 一篇/首个空格Tab切ID/### 归正文
├── internal/render/     # ANSI 渲染层(含测试)
├── internal/server/     # gliderlabs ssh: hostkey/PTY分流/EOF退出/纯文本列表(含测试)
├── internal/theme/      # 3 套主题 palette 结构体 + 切换注册表(默认/aura/冷青)
├── internal/tui/        # 核心重写区(模仿参考布局):
│   ├── app.go           #   根 model: 视图(列表/阅读)状态机, 尺寸, 主题, 用户名注入
│   ├── layout.go        #   边框主区+右 Sidebar 30% 组合渲染(lipgloss 布局)
│   ├── list.go          #   卡片列表(scroll跟随, 摘要)
│   ├── reader.go        #   正文 viewport + 方向键滚动(仅↑↓/j/k, 无Pg/Home/End/g/G)
│   └── sidebar.go       #   STATUS 框(含"主题：<名字>"兜底行) + 提示框
└── articles/articles.md # 文章示例数据
```

依赖白名单：`bubbletea`/`bubbles`/`lipgloss`/`gliderlabs-ssh` 及传递依赖；不用 `testify`；注释中文。
