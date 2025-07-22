# OpenCode 独立安装指南

## 🚀 快速安装

### 1. 构建独立可执行文件

```bash
# 克隆项目（如果还没有）
git clone <repository-url>
cd opencode

# 运行构建脚本
./scripts/build
```

构建完成后，你将得到：

- `build/opencode` - 主启动脚本
- `build/opencode-backend` - 后端服务 (102MB)
- `build/opencode-tui` - TUI 界面 (19MB)
- `dist/opencode-*.tar.gz` - 分发包 (45MB)

### 2. 安装到系统

#### 用户级安装（推荐）

```bash
./scripts/install-system user
```

安装到 `~/.local/bin/`，自动更新 PATH

#### 系统级安装

```bash
./scripts/install-system system
```

安装到 `/usr/local/bin/`，需要 sudo 权限

#### 自定义路径安装

```bash
./scripts/install-system custom /opt/opencode/bin
```

### 3. 验证安装

```bash
# 检查安装状态
./scripts/install-system status

# 测试命令
opencode help
opencode version
```

## 📦 分发安装

如果你要在其他机器上安装：

```bash
# 解压分发包
tar -xzf opencode-*-linux-amd64.tar.gz
cd opencode-*-linux-amd64

# 复制到系统路径
sudo cp opencode* /usr/local/bin/

# 或复制到用户路径
mkdir -p ~/.local/bin
cp opencode* ~/.local/bin/
export PATH="$HOME/.local/bin:$PATH"
```

## 🎯 使用方法

### 基本命令

```bash
# 启动 TUI 界面（默认）
opencode
opencode tui

# 启动后端服务
opencode backend

# 显示帮助
opencode help

# 显示版本
opencode version
```

### 环境变量

```bash
export OPENCODE_CONFIG_DIR="$HOME/.config/opencode"  # 配置目录
export OPENCODE_DATA_DIR="$HOME/.local/share/opencode"  # 数据目录
export OPENCODE_LOG_LEVEL="INFO"  # 日志级别
```

## 🔧 高级配置

### 1. 创建配置文件

在项目根目录创建 `opencode.json`：

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "anthropic/claude-3-5-sonnet-20241022",
  "provider": {
    "anthropic": {
      "options": {
        "cacheControl": true
      }
    }
  }
}
```

### 2. 认证设置

```bash
# 添加 API 密钥
opencode backend auth login

# 列出已配置的提供商
opencode backend auth list
```

## 🗑️ 卸载

```bash
# 卸载用户级安装
./scripts/install-system uninstall-user

# 卸载系统级安装
./scripts/install-system uninstall-system
```

## 🐛 故障排除

### 常见问题

1. **命令未找到**

   ```bash
   # 检查 PATH
   echo $PATH

   # 手动添加到 PATH
   export PATH="$HOME/.local/bin:$PATH"

   # 永久添加到 shell 配置
   echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
   source ~/.bashrc
   ```

2. **权限问题**

   ```bash
   # 确保文件可执行
   chmod +x ~/.local/bin/opencode*
   ```

3. **依赖问题**
   - 独立构建包含所有依赖，无需额外安装
   - 如果遇到问题，检查系统是否支持当前架构

### 日志调试

```bash
# 启用调试日志
export OPENCODE_LOG_LEVEL="DEBUG"
opencode tui

# 查看日志文件
tail -f ~/.local/share/opencode/logs/opencode.log
```

## 📊 性能优化

### 内存使用优化

```bash
# 限制后端内存使用
export NODE_OPTIONS="--max-old-space-size=2048"

# 使用更小的模型
export OPENCODE_SMALL_MODEL="anthropic/claude-3-haiku-20240307"
```

### 缓存配置

```bash
# 清理缓存
rm -rf ~/.cache/opencode/
rm -rf ~/.local/share/opencode/cache/
```

## 🔄 更新

```bash
# 重新构建
git pull
./scripts/build

# 重新安装
./scripts/install-system user --force
```

---

## 📋 构建要求

- **Bun** >= 1.2.0
- **Go** >= 1.21
- **Git**
- **Linux/macOS** (Windows 支持开发中)

## 📁 文件结构

```
build/
├── opencode           # 主启动脚本
├── opencode-backend   # 后端服务 (Bun 编译)
├── opencode-tui       # TUI 界面 (Go 编译)
├── LICENSE
├── README
└── VERSION

dist/
└── opencode-*.tar.gz  # 分发包
```

## 🤝 贡献

如果你在安装过程中遇到问题或有改进建议，请：

1. 检查现有的 Issues
2. 创建新的 Issue 并提供详细信息
3. 提交 Pull Request

---

**作者**: KernelKraze <admin@mail.free-proletariat.dpdns.org>  
**许可证**: MIT  
**项目主页**: https://opencode.ai
