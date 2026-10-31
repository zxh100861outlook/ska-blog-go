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
- 连接后进入三栏 TUI：左主列（文章列表/正文）＋右侧 30% Sidebar（STATUS 框＋提示框），根边框左上角标题 `SKA-BLOG://GO`、底部键位提示条。
- 退出：`Q` 或 `Ctrl+C`。

### 键位表（列表视图）

| 键 | 功能 | 来源（case 分支） |
|---|---|---|
| `↑` / `k` | 选中上移一张卡片 | `internal/tui/app.go` `case "up", "k"` |
| `↓` / `j` | 选中下移一张卡片 | `internal/tui/app.go` `case "down", "j"` |
| `Enter` | 进入当前文章阅读态 | `internal/tui/app.go` `case "enter"` |
| `q` | 退出 | `internal/tui/app.go` `case "q"` |
| `Ctrl+T` | 循环主题（默认→aura→冷青→默认） | `internal/tui/app.go` `case "ctrl+t"` |
| `Ctrl+C` | 退出 | `internal/tui/app.go` `case "ctrl+c"` |
| 滚轮上/下 | 选中上移/下移 | `internal/tui/app.go` `case tea.MouseWheelUp, tea.MouseWheelDown` |

列表滚动跟随：选中卡片始终整块可见，滑出窗口顶/底时窗口整体平移。

搜索态可用 `↑`/`↓` 在命中项间移动光标。

### 键位表（阅读视图）

| 键 | 功能 | 来源（case 分支） |
|---|---|---|
| `PgUp` / `b` | 向上翻一页 | `internal/tui/reader.go` `case "pgup", "b"` |
| `PgDn` / `空格` | 向下翻一页 | `internal/tui/reader.go` `case "pgdown", " "` |
| `j` | 向下滚一行 | `internal/tui/reader.go` `case "j"` |
| `k` | 向上滚一行 | `internal/tui/reader.go` `case "k"` |
| `Home` / `g` | 跳到正文顶部 | `internal/tui/reader.go` `case "home", "g"` |
| `End` / `G` | 跳到正文底部 | `internal/tui/reader.go` `case "end", "G"` |
| `←` / `h` / `Esc` | 返回列表视图 | `internal/tui/app.go` `case "left", "h", "esc"` |
| `q` | 退出 | `internal/tui/reader.go` `case "q"` |
| `Ctrl+T` | 循环主题（列表态/阅读态都可用） | `internal/tui/app.go` `case "ctrl+t"` |
| `Ctrl+C` | 退出 | `internal/tui/app.go` `case "ctrl+c"` |
| 滚轮上/下 | 按加速行数滚动 | `internal/tui/app.go` `case tea.MouseWheelUp, tea.MouseWheelDown` |

阅读态顶部有进度行（`标题 (N%)`），底部提示条列出 `↑↓/PgUp/PgDn 滚动  g/End 行首尾  ←/h/Esc 返回  Ctrl+T 主题  Q/Ctrl+C 退出`。

### 鼠标滚轮连滚加速

`↑↓` 之外的滚轮行为：阅读态下**连续滚轮会加速**。语义（参考 `src/lib/scroll-accel.ts` 的 `fastScroll`，实现见 `internal/tui/reader.go` `accel`）：

- 首次滚轮或间隔 ≥ 300ms：每次滚 1 行（超时重置）。
- 连续区间内每过一拍（50ms）增量翻倍：`1 → 2 → 4 → 8 → 16`，封顶 16 行/次。
- 停手超过 300ms 后回到 1 行/次。

列表态滚轮不做加速，只做单卡移动。

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

1. **交叉编译**（本机，amd64 上产出 linux/arm64 静态二进制）：

   ```bash
   GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o sshblog-arm64 .
   ```

2. **打包 + 上传**（可选 tar 压缩）：

   ```bash
   tar -czf sshblog-arm64.tar.gz sshblog-arm64
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
│   ├── reader.go        #   正文 viewport + 滚动加速
│   └── sidebar.go       #   STATUS 框(含"主题：<名字>"兜底行) + 提示框
└── articles/articles.md # 文章示例数据
```

依赖白名单：`bubbletea`/`bubbles`/`lipgloss`/`gliderlabs-ssh` 及传递依赖；不用 `testify`；注释中文。
