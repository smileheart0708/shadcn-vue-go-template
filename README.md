# shadcn-vue-go-template

一个可直接起步的全栈模板项目：前端使用 Vue 3 + TypeScript + Vite + Tailwind CSS v4，后端使用 Go 1.26+ + SQLite。项目已经内置了登录、注册、用户信息、系统日志、任务列表、主题切换、国际化和前端 mock 等基础能力，适合作为后台管理系统或业务 SPA 的起点。

## 你可以直接得到什么

- 前后端分离的完整项目结构，Go 负责 API 和静态资源托管
- 基于 JWT 的认证流程
- shadcn-vue 风格的组件和页面骨架
- `MSW` 前端 mock，方便不启动后端也能开发登录相关页面
- Vue I18n、Pinia、Vue Router、表格、图表、拖拽等常用能力

## 快速上手

### 1. 安装依赖

```bash
cd web
pnpm install
```

### 2. 启动前端开发

```bash
cd web
pnpm dev
```

默认运行在 `http://localhost:5173`。

### 3. 启动后端开发

先构建前端，再启动 Go 服务：

```bash
cd web
pnpm build

cd ..
go run .
```

后端默认运行在 `http://localhost:8080`。

### 4. Windows 一键构建

```powershell
.\build.ps1
```

会生成带前端资源的单文件可执行程序 `app.exe`。

## 项目结构

```text
.
├── main.go                # Go 入口
├── frontend_embed.go      # 前端资源嵌入
├── internal/              # 后端业务代码
│   ├── auth/
│   ├── config/
│   ├── database/
│   ├── httpapi/
│   ├── logging/
│   └── users/
└── web/                   # Vue 前端
    ├── src/
    │   ├── components/
    │   ├── layouts/
    │   ├── lib/
    │   ├── locales/
    │   ├── router/
    │   ├── stores/
    │   └── views/
    └── dist/              # 构建产物
```

## 常用命令

### 前端

```bash
cd web
pnpm build         # 生产构建
pnpm lint          # ESLint 检查
pnpm typecheck     # TypeScript 检查
pnpm format:check  # Prettier 检查
```

### 后端

```bash
go test ./...              # 运行全部测试
go build -tags=go_json .    # 构建后端
```

## 配置说明

常用环境变量：

- `PORT`：后端端口，默认 `8080`
- `DATA_DIR`：SQLite 数据目录，默认 `./.data`
- `DB_NAME`：数据库文件名，默认 `data.db`
- `JWT_SECRET`：JWT 密钥；不配置时会自动生成并持久化
- `VITE_API_MOCKING`：启用前端 auth mock

前端 mock 模式只覆盖登录和当前用户接口，适合 UI 联调。

## 适合怎么改

如果你要基于这个模板继续开发，通常从这几处开始：

- 新页面放到 `web/src/views/`
- 可复用组件放到 `web/src/components/`
- API 封装放到 `web/src/lib/api/`
- 后端接口放到 `internal/httpapi/`
- 用户和认证逻辑放到 `internal/users/` 和 `internal/auth/`
