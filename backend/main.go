package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ===========================
// 1. 配置区域 (保持不变)
// ===========================
const (
	DB_DSN           = "root:rootpassword@tcp(192.168.31.143:3307)/edu_platform?charset=utf8mb4&parseTime=True&loc=Local"
	MINIO_ENDPOINT   = "192.168.31.143:9000"
	MINIO_ACCESS_KEY = "admin"
	MINIO_SECRET_KEY = "password123"
	MINIO_USE_SSL    = false
	BUCKET_PICTURES  = "pictures"
	BUCKET_VIDEOS    = "videos"
	JWT_SECRET       = "my_super_secret_key_2026"
)

// ===========================
// 2. 升级后的数据模型
// ===========================

type User struct {
	gorm.Model
	Username string `gorm:"unique;not null" json:"username"`
	Password string `json:"-"`
	Role     string `json:"role"` // student, teacher, admin
	Avatar   string `json:"avatar"`
}

type Course struct {
	gorm.Model
	Title       string  `json:"title"`
	Description string  `json:"description"`
	TeacherID   uint    `json:"teacher_id"`
	CoverImage  string  `json:"cover_image"`
	VideoURL    string  `json:"video_url"`
	Price       float64 `json:"price"`
	
	// --- 新增字段 ---
	Category    string  `json:"category"`    // 课程分类 (如: frontend, backend, ai)
	ViewCount   int     `json:"view_count"`  // 浏览量/热度
	// ----------------
	
	Homeworks []Homework `gorm:"foreignKey:CourseID" json:"homeworks"`
}

// Enrollment 选课记录/学习进度
type Enrollment struct {
	gorm.Model
	UserID    uint    `json:"user_id"`
	CourseID  uint    `json:"course_id"`
	Progress  float64 `json:"progress"` // 学习进度 0-100
	IsFinish  bool    `json:"is_finish"`
	Course    Course  `gorm:"foreignKey:CourseID" json:"course"`
}

// Homework 作业与提交记录
type Homework struct {
	gorm.Model
	CourseID  uint   `json:"course_id"`
	StudentID uint   `json:"student_id"`
	Content   string `json:"content"` // 学生提交的内容
	Score     int    `json:"score"`   // 分数 (0代表未批改)
	Comment   string `json:"comment"` // 老师评语
}

var db *gorm.DB
var minioClient *minio.Client

// ===========================
// 3. 辅助函数 (Init, JWT)
// ===========================

func initDB() {
	var err error
	db, err = gorm.Open(mysql.Open(DB_DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}
	// 自动迁移所有表
	db.AutoMigrate(&User{}, &Course{}, &Enrollment{}, &Homework{})
	log.Println("✅ 数据库表结构已更新")
}

func initMinIO() {
	var err error
	minioClient, err = minio.New(MINIO_ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(MINIO_ACCESS_KEY, MINIO_SECRET_KEY, ""),
		Secure: MINIO_USE_SSL,
	})
	if err != nil {
		log.Fatalf("❌ MinIO 连接失败: %v", err)
	}
	// 确保桶存在 (省略重复代码，假设桶已存在)
}

func GenerateToken(userID uint, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWT_SECRET))
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token格式错误"})
			return
		}
		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			return []byte(JWT_SECRET), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token无效"})
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		c.Set("userID", uint(claims["user_id"].(float64)))
		c.Set("role", claims["role"].(string))
		c.Next()
	}
}

// ===========================
// 4. 业务逻辑 Handlers
// ===========================

// 注册 & 登录 (保持之前逻辑，略微精简)
func RegisterHandler(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	user.Password = string(hashedPwd)
	if user.Role == "" { user.Role = "student" }
	if err := db.Create(&user).Error; err != nil {
		c.JSON(500, gin.H{"error": "注册失败"})
		return
	}
	c.JSON(200, gin.H{"message": "注册成功"})
}

func LoginHandler(c *gin.Context) {
	var input User
	c.ShouldBindJSON(&input)
	var user User
	if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(401, gin.H{"error": "用户不存在"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)) != nil {
		c.JSON(401, gin.H{"error": "密码错误"})
		return
	}
	token, _ := GenerateToken(user.ID, user.Role)
	c.JSON(200, gin.H{"token": token, "role": user.Role, "username": user.Username, "user_id": user.ID})
}

// --- 课程相关 ---

func CreateCourseHandler(c *gin.Context) {
	var course Course
	if err := c.ShouldBindJSON(&course); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	// 设置默认初始热度
	course.ViewCount = 0
	db.Create(&course)
	c.JSON(200, gin.H{"message": "发布成功", "data": course})
}

func ListCoursesHandler(c *gin.Context) {
	var courses []Course
	category := c.Query("category")
	sort := c.Query("sort") // sort=hot 代表热门

	tx := db.Model(&Course{})

	// 1. 分类筛选
	if category != "" && category != "all" {
		tx = tx.Where("category = ?", category)
	}

	// 2. 排序逻辑 (默认按时间倒序，热门按浏览量倒序)
	if sort == "hot" {
		tx = tx.Order("view_count desc").Limit(5) // 只取前5个热门
	} else {
		tx = tx.Order("created_at desc")
	}

	tx.Find(&courses)
	c.JSON(200, gin.H{"data": courses})
}

// GetCourseDetailHandler 获取课程详情（包含是否已选课信息）
func GetCourseDetailHandler(c *gin.Context) {
	courseID := c.Param("id")
	var course Course
	if err := db.First(&course, courseID).Error; err != nil {
		c.JSON(404, gin.H{"error": "课程不存在"})
		return
	}
	db.Model(&course).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))
	// ---------------------------------------
	// 如果用户登录了，检查是否已选课
	isEnrolled := false
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// 简单解析一下 Token 拿 UserID，实际可以用中间件
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 {
			token, _ := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) { return []byte(JWT_SECRET), nil })
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				uid := uint(claims["user_id"].(float64))
				var count int64
				db.Model(&Enrollment{}).Where("user_id = ? AND course_id = ?", uid, course.ID).Count(&count)
				if count > 0 { isEnrolled = true }
			}
		}
	}

	c.JSON(200, gin.H{"course": course, "is_enrolled": isEnrolled})
}

// --- 学习与作业相关 ---

// EnrollHandler 学生选课
func EnrollHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct { CourseID uint `json:"course_id"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	
	// 检查是否重复选课
	var count int64
	db.Model(&Enrollment{}).Where("user_id = ? AND course_id = ?", userID, req.CourseID).Count(&count)
	if count > 0 {
		c.JSON(400, gin.H{"error": "已加入该课程"})
		return
	}

	enroll := Enrollment{UserID: userID, CourseID: req.CourseID, Progress: 0}
	db.Create(&enroll)
	c.JSON(200, gin.H{"message": "加入课程成功！"})
}

// GetMyCoursesHandler 获取“我的课程”列表
func GetMyCoursesHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var enrolls []Enrollment
	// 预加载 Course 信息
	db.Preload("Course").Where("user_id = ?", userID).Find(&enrolls)
	c.JSON(200, gin.H{"data": enrolls})
}

// SubmitHomeworkHandler 提交作业
func SubmitHomeworkHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var hw Homework
	if err := c.ShouldBindJSON(&hw); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	hw.StudentID = userID
	// 简单的覆盖更新逻辑：如果交过，就更新内容
	var exist Homework
	if err := db.Where("course_id = ? AND student_id = ?", hw.CourseID, userID).First(&exist).Error; err == nil {
		exist.Content = hw.Content
		db.Save(&exist)
	} else {
		db.Create(&hw)
	}
	c.JSON(200, gin.H{"message": "作业提交成功"})
}

// GetHomeworkHandler 获取某课程的作业信息
func GetHomeworkHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	courseID := c.Query("course_id")
	var hw Homework
	if err := db.Where("course_id = ? AND student_id = ?", courseID, userID).First(&hw).Error; err != nil {
		c.JSON(200, gin.H{"exists": false}) // 没交过
		return
	}
	c.JSON(200, gin.H{"exists": true, "data": hw})
}

// UploadHandler (保持不变)
func UploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil { c.JSON(400, gin.H{"error": "No file"}); return }
	bucket := BUCKET_PICTURES
	if filepath.Ext(file.Filename) == ".mp4" { bucket = BUCKET_VIDEOS }
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
	src, _ := file.Open(); defer src.Close()
	minioClient.PutObject(context.Background(), bucket, filename, src, file.Size, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	c.JSON(200, gin.H{"url": fmt.Sprintf("http://%s/%s/%s", MINIO_ENDPOINT, bucket, filename)})
}

// ===========================
// 5. Main 入口
// ===========================

func main() {
	initDB()
	initMinIO()

	r := gin.Default()
	// CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" { c.AbortWithStatus(204); return }
		c.Next()
	})

	api := r.Group("/api/v1")
	{
		api.POST("/register", RegisterHandler)
		api.POST("/login", LoginHandler)
		api.GET("/courses", ListCoursesHandler)
		api.GET("/courses/:id", GetCourseDetailHandler) // 课程详情

		// 需登录接口
		auth := api.Group("/")
		auth.Use(AuthMiddleware())
		{
			auth.POST("/upload", UploadHandler)
			auth.POST("/courses", CreateCourseHandler) // 老师发课
			
			// 学生学习相关
			auth.POST("/enroll", EnrollHandler)       // 加入课程
			auth.GET("/my-courses", GetMyCoursesHandler) // 个人中心课程
			auth.POST("/homework", SubmitHomeworkHandler) // 交作业
			auth.GET("/homework", GetHomeworkHandler) // 看作业状态
		}
	}

	r.Run(":8080")
}