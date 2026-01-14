# Telegram Translation Bot

一个集成 OpenAI 的 Telegram 机器人，支持自定义预设、多轮对话上下文管理和完善的用户权限系统。

## 功能特点

- **架构优化**：采用依赖注入（Dependency Injection）设计，代码结构清晰，易于扩展和维护。
- **高性能 OpenAI 集成**：
  - 复用 HTTP 客户端连接（Keep-Alive），提升响应速度。
  - 支持完整的对话上下文（Context）传递，实现丝滑的多轮对话。
- **智能上下文管理**：
  - 使用 Redis List 存储对话历史，规避并发写入冲突。
  - 自动长度控制：基于字符数（Rune Count）智能裁剪过长历史，确保不触发 API 限制。
- **安全与限流**：
  - **频率限制**：内置每分钟消息限流机制，保护 API 额度不被滥用。
  - **白名单系统**：完善的用户授权与有效期管理，支持多管理员。
- **自定义预设**：支持通过配置文件自定义 System Prompt 和快捷按钮。
- **Docker 部署**：一键式环境搭建，支持热加载开发模式。

## 命令列表

### 普通用户命令
- `/start` - 开始使用机器人，显示功能选项
- `/help` - 获取帮助信息
- `/about` - 关于我们
- `/clear` - 清空当前对话历史和预设
- `/expiry` - 查看您的使用权限有效期
- `/id` - 获取您的用户ID
- `/chinese_to_japanese` 等 - 预设翻译模式切换（支持自定义）

### 管理员命令
- `/adduser <用户ID> [天数]` - 添加用户到白名单
- `/deleteuser <用户ID>` - 从白名单删除用户
- `/extend <用户ID> <天数]` - 延长用户使用期限
- `/checkuser [用户ID]` - 查看用户列表或指定用户状态

## 技术栈

- **语言**: Go 1.22+
- **数据库**: PostgreSQL 14+ (用户权限)
- **缓存**: Redis (对话上下文、频率限制)
- **框架**: `telegram-bot-api/v5`, `gorm`, `go-redis/v8`

## 环境要求

- Go 1.22+
- PostgreSQL
- Redis
- Docker & Docker Compose (推荐)

## 配置说明

### 环境变量
创建 `.env` 文件，参考 `.env.example`：
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
OPENAI_MODEL=gpt-4o  # 推荐使用

# Admin
ADMIN_USER_IDS=12345678,98765432
```

## 部署说明

### Docker 部署 (推荐)
1. 克隆仓库并进入目录
2. 配置 `.env` 文件
3. 启动服务：
   ```bash
   docker-compose up -d
   ```

### 手动部署
...
```

## 进阶功能：使用外部数据库

如果你已经有现成的 PostgreSQL 数据库，不想在 Docker 中启动新的：

1. **修改 `.env`**：将 `DB_HOST` 指向你的外部数据库地址，并填写正确的端口、用户名和密码。
2. **选择性启动**：只启动机器人服务和 Redis：
   ```bash
   docker-compose up -d tg-bot tg_go_redis
   ```
   或者直接注释掉 `docker-compose.yml` 中的 `tg_go_postgres` 部分。


## 项目结构

- `main.go`: 程序入口，负责依赖注入与生命周期管理。
- `handlers/`:
  - `init.go`: 核心 `Handler` 结构定义。
  - `message.go`: 文本消息处理、限流与上下文逻辑。
  - `command.go`: 通用与预设命令逻辑。
  - `admin.go`: 管理员特权指令。
  - `callback.go`: 按钮回调处理。
- `openai/`: 封装 OpenAI API 调用与连接池。
- `models/`: GORM 数据库模型与权限逻辑。
- `config/`: 配置文件与环境变量加载。

## 注意事项

1. **频率限制**：默认限制为每位用户 10 条消息/分钟，可在 `handlers/message.go` 中修改。
2. **上下文过期**：对话历史在 Redis 中默认保留 30 分钟。
3. **数据迁移**：启动时会自动执行 GORM AutoMigrate。

## 许可证

[MIT License](LICENSE)
