# 🎓 在线教育平台 (Online Education Platform)

> **本项目采用“本地服务器 + 远程开发”的模式进行协作。**
> 
> * **服务器 (Server):** 部署在 [队长名字] 的电脑上，运行 Docker 基础设施 (MySQL, MinIO, Redis, Nginx)。
> * **开发者 (You):** 在自己的电脑上编写后端代码，连接服务器的数据库进行调试。

## 🚀 1. 核心连接信息 (Server Info)

请将这些配置写入你的代码配置文件或环境变量中。

| 服务 | IP 地址 (Host) | 端口 | 账号 | 密码 | 备注 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **MySQL** | `192.168.31.143` | **3307** | `root` | `rootpassword` | **⚠️ 注意外部端口是 3307** |
| **MinIO (S3)** | `192.168.31.143` | **9000** | `admin` | `password123` | `SSL: false` |
| **Redis** | `192.168.31.143` | **6379** | - | - | 无密码 |
| **Nginx (API)**| `192.168.31.143` | **80** | - | - | 前端请求入口 |

**管理后台快捷入口：**
* 📊 **数据库管理 (Adminer):** [http://192.168.31.143:8081](http://192.168.31.143:8081) (系统选 MySQL，服务器填 `mysql`)
* 🗄️ **对象存储后台 (MinIO):** [http://192.168.31.143:9001](http://192.168.31.143:9001)

---

## 💻 2. 团队开发工作流 (必读)

为了防止代码冲突和环境混乱，请严格遵守以下流程。

### A. 队友：如何提交代码 (Push)

作为后端开发人员，你的日常操作步骤如下：

1.  **开始工作前，先拉取最新代码：**
    ```bash
    git pull origin main
    ```
2.  **编写代码：**
    * 在 `backend/` 目录下编写 Go 代码。
    * **连接配置：** 确保你的代码连接的是 `192.168.31.143` (不要写 localhost)。
    * *建议：* 使用 `go run main.go` 在本地测试，确保逻辑跑通。
3.  **提交代码：**
    ```bash
    # 1. 添加修改文件
    git add .

    # 2. 提交并写明修改内容 (例如：完成用户注册接口)
    git commit -m "feat: finish user register api"

    # 3. 推送到 GitHub
    git push origin main
    ```
4.  **通知管理员：** 代码推送成功后，请在群里喊一声：“代码已上传，求部署！”

---

### B. 管理员：如何部署更新 (Deploy)

作为服务器维护者（队长），当队友通知更新后，你需要执行：

1.  **进入项目目录：**
    ```powershell
    cd G:\OnlineEducationPaltform
    ```
2.  **拉取最新代码：**
    ```powershell
    git pull origin main
    ```
    *(如果有冲突，请先解决冲突)*
3.  **重启后端容器 (使新代码生效)：**
    ```powershell
    docker-compose restart backend
    ```
4.  **验证：**
    查看日志确认无报错：
    ```powershell
    docker logs -f edu_backend
    ```

---

## 🛠️ 3. 开发环境配置示例 (Go)

建议在你的 `main.go` 或配置文件中使用以下连接字符串：

```go
package main

// ... imports

func main() {
    // 数据库连接 (DSN)
    // 格式: user:pass@tcp(IP:PORT)/dbname...
    dsn := "root:rootpassword@tcp(192.168.31.143:3307)/edu_platform?charset=utf8mb4&parseTime=True&loc=Local"
    
    // MinIO 配置
    minioEndpoint := "192.168.31.143:9000"
    minioAccessKey := "admin"
    minioSecretKey := "password123"
    useSSL := false
    
    // ... 业务逻辑
}

❓ 4. 常见问题 (Troubleshooting)
Q: 连不上数据库或 MinIO？

检查 IP: 确认队长的 IP 还是不是 192.168.31.143 (有时路由器重启会变)。

检查网络: 确认你和服务器连在同一个 WiFi 下。

Ping 测试: 打开终端输入 ping 192.168.31.143 看通不通。

Q: Git Push 失败？

提示 "Updates were rejected": 说明有人比你先提交了。请先执行 git pull origin main 合并代码，然后再 Push。

提示 "Connection refused": 网络问题，尝试开启加速器或多试几次。

Q: 图片上传成功但访问不了？

确认 MinIO 的 Bucket 权限是否设为 Public。

访问链接应该是：http://192.168.31.143:9000/桶名/文件名。
