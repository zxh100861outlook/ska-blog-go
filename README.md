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
- 连接后进入 TUI（Charm house style，soft-serve/glow 式）：顶部一行 header 无边框——左 `SKA·BLOG` 品牌字（主色粗体），右对齐 `访客：<user> · 文章 N 篇`；header 下一条细分割线。主体为左主列（glow 式行列表/正文）＋右侧 30% Sidebar，两栏间一条竖分割线（仅侧栏展开时画）。侧栏为无边框 `STATUS` / `KEYS` 两个小节（标题大写小字灰、内容缩进两格，键位逐行列出并附"主题：X"与"侧栏：开/收"）。`Tab` 可收起/展开侧栏，收起后主列占满全宽（无竖分割线）。
- **复制**：终端**没有接管鼠标**（不发送 `?1000`/`?1006` 上报），直接用鼠标**拖拽选中**即可复制，无需 Ctrl+C 手势。
- 退出：`Q` 或 `Ctrl+C`。

### 键位表（列表视图）

| 键 | 功能 | 来源（case 分支） |
|---|---|---|
| `↑` / `k` | 选中上移一个条目 | `internal/tui/app.go` `case "up", "k"` |
| `↓` / `j` | 选中下移一个条目 | `internal/tui/app.go` `case "down", "j"` |
| `Enter` | 进入当前文章阅读态 | `internal/tui/app.go` `case "enter"` |
| `/` | 进入搜索态（实时过滤） | `internal/tui/app.go` `case "/"` |
| `:` | 进入编号跳转态 | `internal/tui/app.go` `case ":"` |
| `Tab` | 收起/展开右侧 Sidebar | `internal/tui/app.go` `case "tab"` |
| `Ctrl+T` | 循环主题（默认→aura→冷青→默认） | `internal/tui/app.go` `case "ctrl+t"` |
| `Q` / `Ctrl+C` | 退出 | `internal/tui/app.go` `case "ctrl+c"` |

列表为 glow 式**行列表（无边框盒）**：每篇两行——首行 `▍ <id> · <标题>`（选中：整行淡背景色块 + 行首 `▍` 块状光标 + 标题主色加粗；未选：前景 TextMuted），次行 `<id>·N段·约M分钟 · <摘要>`（更暗一档、同缩进两格），条目间空一行。滚动跟随：选中条目始终整块可见，滑出窗口顶/底时窗口整体平移。

### 键位表（阅读视图）

| 键 | 功能 | 来源（case 分支） |
|---|---|---|
| `↑` / `k` | 向上滚一行 | `internal/tui/reader.go` `case "up", "k"` |
| `↓` / `j` | 向下滚一行 | `internal/tui/reader.go` `case "down", "j"` |
| `←` / `h` / `Esc` | 返回列表视图 | `internal/tui/app.go` `case "left", "h", "esc"` |
| `Tab` | 收起/展开右侧 Sidebar | `internal/tui/app.go` `case "tab"` |
| `Ctrl+T` | 循环主题（列表态/阅读态都可用） | `internal/tui/app.go` `case "ctrl+t"` |
| `Q` / `Ctrl+C` | 退出 | `internal/tui/app.go` `case "ctrl+c"` |

阅读态顶部有进度行（`标题 (N%)`）；正文用 **glamour** 渲染（标题层级/代码块背景，按主列宽折行，失败时回退 `internal/render`）。滚动**只保留方向键**（`↑↓`/`j`/`k`），`PgUp`/`PgDn`/`Space`/`Home`/`End`/`g`/`G` 一律不响应。键位提示在侧栏 `KEYS` 小节按当前视图（列表/阅读/搜索/跳转）动态列出。

### 搜索与跳转

- **搜索**：列表视图按 `/` 进入搜索态，主列顶部出现输入行。键入时**实时过滤**——按标题＋正文＋ID 做不区分大小写的子串匹配，列表只显示命中项且焦点在首个命中。`Enter` 直接打开焦点文章阅读，`Esc` 取消搜索并恢复全量，`退格` 删除上一个字符（中文按字符计）。
- **编号跳转**：列表视图按 `:` 进入跳转态，输入文章 ID（先精确、后前缀匹配）。`Enter` 命中则直达阅读；未命中在输入行后显示"无此文"且保留输入；`Esc` 取消。
- 输入行为手写 rune 级实现（不依赖 `bubbles/textinput`），中文按客户端送来的字节解码累积。

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
| `-render` | `""` | 调试：渲染指定 ID 的文章 ANSI 文本后退出（按 80 列折行），如 `go run . -render wj2` |
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

- **终端全黑白、切主题看不出变化**：多为客户端 `TERM` 与服务端探测不兼容（服务端进程无 `TERM` 时按 stdout 探测，守护进程/docker 下可能退化为无色档）。此时看侧栏 `KEYS` 小节的 **`· 主题：<名字>`** 一行——它是纯文字兜底，无论是否有色都会随 `Ctrl+T` 变化，可用它确认主题切换生效。若需恢复颜色，让客户端以 `TERM=xterm-256color` 或 `COLORTERM=truecolor` 连接（如 `tmux` 会话内 `ssh -tt`），或在板子上用 systemd/tmux 赋予服务端一个真实 TTY 与 `TERM`。
- **白屏 / 一直 `loading...`**：`View()` 在收到 `tea.WindowSizeMsg` 前返回 `"loading..."` 占位，绝不返回空串。排查入口：
  1. 确认带 PTY 连接（`ssh -tt -p 2222 主机`，非 `-T`）。
  2. 确认终端尺寸可上报（tmux/screen 正常；纯管道/无 PTY 会走纯文本列表分支）。
  3. 看服务端进程 stdout 是否有异常（`-local` 无 TTY 会报 `could not open a new TTY`）。
- **`ssh -T` 与 TUI 行为不同**：`-T` 无 PTY → 纯文本列表；进 TUI 必须带 PTY（`-tt`）。
- **终端拖拽选中不能复制**：本项目已**不接管鼠标**（`server.go` 未启用 `WithMouseCellMotion`，不发送 `?1000`/`?1006` 上报），因此终端的拖拽选中复制应原生可用。若仍不能复制，通常是终端自身选中模式/`mouse` 选项问题（如 tmux 需要 `set -g mouse off` 或按住 `Shift` 拖拽），与本服务端无关。
- **滚动失效**：阅读态滚动**只保留方向键**（`↑↓`/`j`/`k`），`PgUp`/`PgDn`/`Space`/`Home`/`End`/`g`/`G` 与鼠标滚轮**故意不响应**（交互纠偏：滚轮会破坏拖拽复制，Pg 类键位已裁剪）。若文章很短（正文一屏内放得下），方向键自然无滚动空间，属预期行为。

---

## 项目结构

与 `SPEC.md` 的文件职责表核对一致：

```
ska-blog-go/
├── main.go              # flag: -addr/-articles/-guest(默认true)/-list/-render/-local; 模式分流(默认server, -local调试)
├── internal/content/    # articles.md 解析层(含测试): # 跳过/## 一篇/首个空格Tab切ID/### 归正文
├── internal/render/     # ANSI 渲染层(含测试) —— 列表摘要/阅读兜底/-render/纯文本列表用
├── internal/server/     # gliderlabs ssh: hostkey/PTY分流/EOF退出/纯文本列表(含测试)
├── internal/theme/      # 3 套主题 palette(含 glow 衍生色: SelBg/ListTitle/Meta/Divider) + 切换注册表
├── internal/tui/        # 核心重写区(Charm house style):
│   ├── app.go           #   根 model: 视图(列表/阅读)状态机, 尺寸, 主题, 用户名注入, 搜索/跳转/Tab
│   ├── layout.go        #   soft-serve 布局: header+分割线+两栏(竖分割线), 无整周包边
│   ├── list.go          #   glow 式行列表(无边框盒, ▍光标, 选中背景色块)
│   ├── reader.go        #   正文 viewport(glamour 渲染+internal/render 兜底, 方向键滚动)
│   └── sidebar.go       #   无边框 STATUS/KEYS 小节(键位+主题/侧栏状态行)
└── articles/articles.md # 文章示例数据
```

依赖白名单：`bubbletea`/`bubbles`/`lipgloss`/`glamour`/`gliderlabs-ssh` 及传递依赖；不用 `testify`；注释中文。
