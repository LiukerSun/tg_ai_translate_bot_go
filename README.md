# Telegram Translation Bot

一个集成 OpenAI 的 Telegram 机器人，支持自定义预设。

## 功能特点

- 支持自定义预设提示词
- 预设模式切换
- 用户白名单管理系统
- 用户有效期管理
- 多管理员支持
- 对话上下文管理
- Docker 部署支持

## 命令列表

### 普通用户命令
- `/start` - 开始使用机器人，显示功能选项
- `/help` - 获取帮助信息
- `/about` - 关于我们
- `/clear` - 清空当前对话历史和预设
- `/expiry` - 查看您的使用权限有效期
- `/id` - 获取您的用户ID

### 管理员命令
- `/adduser <用户ID> [天数]` - 添加用户到白名单
- `/deleteuser <用户ID>` - 从白名单删除用户
- `/extend <用户ID> <天数>` - 延长用户使用期限
- `/checkuser [用户ID]` - 查看用户列表或指定用户状态

### 预设模式
- `/chinese_to_japanese` - 切换到中译日模式
- `/japanese_to_chinese` - 切换到日译中模式
- `/russian_mode` - 切换到俄语翻译模式

## 环境要求

- Go 1.22+
- PostgreSQL 14+
- Redis
- Docker & Docker Compose (可选)

## 配置说明

### 环境变量
创建 `.env` 文件，包含以下配置：
```env
# Database
DB_HOST=postgres
DB_NAME=tg_bot
DB_USER=your_db_user
DB_PASSWORD=your_db_password
DB_PORT=5432
DB_SSLMODE=disable

# Redis
REDIS_HOST=tg_go_redis
REDIS_PORT=6379

# Telegram
TELEGRAM_BOT_TOKEN=your_bot_token

# OpenAI
OPENAI_API_URL=your_api_url
OPENAI_API_KEY=your_api_key
OPENAI_MODEL=gpt-3.5-turbo

# Admin
ADMIN_USER_IDS=admin_id1,admin_id2
```


## 部署说明

### Docker 部署
1. 克隆仓库
```bash
git clone https://github.com/yourusername/tg-bot-go.git
cd tg-bot-go
```
2. 配置环境变量
```bash
cp .env.example .env
#编辑 .env 文件，填入必要的配置信息
```
3. 启动服务
```bash
docker-compose up -d
```
### 手动部署
1. 安装依赖
bash
go mod download

2. 编译
```bash
go build -o tg-bot-go main.go
```

3. 运行
```bash
./tg-bot-go
```

## 开发模式

使用 `-dev` 标志启动开发模式：
```bash
go run main.go -dev
```


开发模式会加载 `.env.dev` 文件中的配置。

## 预设模式

机器人支持两种类型的按钮：

### Inline Keyboard 按钮
- 显示在消息中的按钮
- 用于选择预设模式
- 在 `/start` 和 `/clear` 命令后显示

### Reply Keyboard 按钮
- 显示在输入框上方的持久按钮
- 包含常用命令如 `/clear`
- 方便用户快速访问常用功能

### 自定义按钮和预设

可以通过修改 `config/presets.toml` 文件来自定义预设按钮：

```toml
[[items]]
button = "中译日"    # 按钮显示文本
command = "/chinese_to_japanese"    # 按钮命令
content = """
你是一个翻译引擎，负责将输入的中文文本翻译成日语。要求如下：
1. 忠实于原文意思，逐字逐句翻译，不添加、不删减或改写内容。
2. 翻译结果必须符合日语的自然表达方式，贴近日常对话。
3. 确保语法正确，并传递原文的语气和情感。

示例：
- 输入："你好" -> 输出："こんにちは"
- 输入："你是谁？" -> 输出："あなたは誰ですか？"
"""
```

添加新的预设：
1. 在 `presets.toml` 中添加新的 `[[items]]` 配置
2. 设置按钮显示文本 (`button`)
3. 设置命令名称 (`command`)
4. 编写预设提示词 (`content`)
5. 重启机器人使配置生效

注意事项：
- 命令名称必须以 `/` 开头



## 数据持久化

- PostgreSQL 数据存储在 `tg_go_postgres_data` 卷中
- Redis 数据存储在 `tg_go_redis_data` 卷中
- 日志文件存储在 `./logs` 目录

## 注意事项

1. 首次运行时会自动创建数据库表结构
2. 确保在启动前已正确配置管理员ID
3. Redis 用于存储对话上下文，重启后会清空
4. 用户有效期到期后需要管理员手动延期
5. 建议定期备份数据库

## 许可证

[MIT License](LICENSE)
