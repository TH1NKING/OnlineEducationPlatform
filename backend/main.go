package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql" // 导入 MySQL 驱动
)

var db *sql.DB

func main() {
	// 1. 配置数据库连接信息
	// 注意：这里的 Host 是 "mysql" (容器服务名)，不是 localhost
	// 因为在 Docker 网络内部，容器之间通过服务名互相访问
	dbUser := "root"
	dbPassword := "rootpassword"
	dbHost := os.Getenv("DB_HOST") // 从 docker-compose 环境变量读取
	if dbHost == "" {
		dbHost = "mysql" // 默认值
	}
	dbPort := "3306" // 容器内部端口是 3306
	dbName := "edu_platform"

	// 拼接 DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	// 2. 连接数据库 (带重试机制，防止数据库还没启动好后端就崩了)
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping() // 真正尝试建立连接
			if err == nil {
				log.Println("✅ 成功连接到 MySQL 数据库！")
				break
			}
		}
		log.Printf("⚠️ 等待数据库启动... (%d/10) 错误: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("❌ 无法连接数据库: %v", err)
	}

	// 3. 设置简单的 API 路由
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "后端服务运行正常！DB连接状态: %v", db.Stats().OpenConnections)
	})

	// 测试接口：查询用户表
	http.HandleFunc("/api/users", handleUsers)

	log.Println("🚀 后端服务启动在 :8080")
	http.ListenAndServe(":8080", nil)
}

// 一个简单的接口，查询数据库里的用户
func handleUsers(w http.ResponseWriter, r *http.Request) {
	// 简单查询一下 users 表
	rows, err := db.Query("SELECT username, role FROM users")
	if err != nil {
		http.Error(w, "查询失败: "+err.Error(), 500)
		return
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var u, role string
		rows.Scan(&u, &role)
		users = append(users, fmt.Sprintf("%s (%s)", u, role))
	}

	fmt.Fprintf(w, "数据库中的用户: %v", users)
}
