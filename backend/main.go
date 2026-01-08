package main

import (
	"context"
	"errors"
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
// 1. 配置区域
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
// 2. 数据模型
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
	Category    string  `json:"category"`
	ViewCount   int     `json:"view_count"`
	Outline     string  `json:"outline" gorm:"type:text"`      // 存储 JSON 字符串: [{"title":"第一章","desc":"..."}, ...]
	HomeworkReq string  `json:"homework_req" gorm:"type:text"`
	Homeworks   []Homework `gorm:"foreignKey:CourseID" json:"homeworks"`
}

type Question struct {
	gorm.Model
	CourseID   uint   `json:"course_id"`
	StudentID  uint   `json:"student_id"`
	Student    User   `gorm:"foreignKey:StudentID" json:"student"` // 关联学生信息
	Content    string `json:"content"`                             // 问题内容
	Answer     string `json:"answer"`                              // 老师回复
	TeacherID  uint   `json:"teacher_id"`                          // 回复的老师ID
	IsAnswered bool   `json:"is_answered"`                         // 是否已回复
}

type Enrollment struct {
	gorm.Model
	UserID    uint    `json:"user_id"`
	CourseID  uint    `json:"course_id"`
	Progress  float64 `json:"progress"`
	IsFinish  bool    `json:"is_finish"`
	Course    Course  `gorm:"foreignKey:CourseID" json:"course"`
}

type Homework struct {
	gorm.Model
	CourseID  uint   `json:"course_id"`
	StudentID uint   `json:"student_id"`
	Content   string `json:"content"`
	Score     int    `json:"score"`
	Comment   string `json:"comment"`
}

var db *gorm.DB
var minioClient *minio.Client

// ===========================
// 3. 初始化与工具函数
// ===========================

func initDB() {
	var err error
	db, err = gorm.Open(mysql.Open(DB_DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}
	
	// 自动迁移
	db.AutoMigrate(&User{}, &Course{}, &Enrollment{}, &Homework{}, &Question{})

	// --- 修复逻辑：更加健壮的管理员初始化 ---
	var admin User
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	
	// 使用 Unscoped() 以查找可能被软删除的记录，防止唯一键冲突
	err = db.Unscoped().Where("username = ?", "admin").First(&admin).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 确实不存在，创建新账号
		log.Println("⚡️ 未找到管理员，正在创建...")
		adminUser := User{
			Username: "admin",
			Password: string(hashedPwd),
			Role:     "admin",
		}
		if createErr := db.Create(&adminUser).Error; createErr != nil {
			log.Printf("❌ 创建管理员失败: %v", createErr)
		} else {
			log.Println("✅ 管理员创建成功 -> 账号: admin / 密码: 123456")
		}
	} else {
		// 存在（包括被软删除的），强制恢复并重置密码
		log.Println("⚠️ 检测到已有管理员，正在重置状态和密码...")
		
		// 恢复被软删除的记录
		if admin.DeletedAt.Valid {
			db.Unscoped().Model(&admin).Update("deleted_at", nil)
		}

		// 更新密码和角色
		db.Model(&admin).Updates(map[string]interface{}{
			"password": string(hashedPwd),
			"role":     "admin",
		})
		log.Println("✅ 管理员重置成功 -> 账号: admin / 密码: 123456")
	}
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token无效或已过期"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token解析失败"})
			return
		}

		c.Set("userID", uint(claims["user_id"].(float64)))
		c.Set("role", claims["role"].(string))
		c.Next()
	}
}

// ===========================
// 4. Handler 逻辑
// ===========================

func RegisterHandler(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if input.Username == "" || input.Password == "" {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	if input.Role == "admin" {
		c.JSON(403, gin.H{"error": "无法注册管理员"})
		return
	}
	role := input.Role
	if role == "" { role = "student" }

	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	user := User{
		Username: input.Username,
		Password: string(hashedPwd),
		Role:     role,
	}
	
	if err := db.Create(&user).Error; err != nil {
		c.JSON(500, gin.H{"error": "用户名已存在"})
		return
	}
	c.JSON(200, gin.H{"message": "注册成功"})
}

func LoginHandler(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	if input.Username == "" || input.Password == "" {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}

	var user User
	// 使用 Unscoped 以防之前被软删除导致无法登录
	if err := db.Unscoped().Where("username = ?", input.Username).First(&user).Error; err != nil {
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

// --- 公开接口 ---

func ListCoursesHandler(c *gin.Context) {
	var courses []Course
	category := c.Query("category")
	sort := c.Query("sort")

	tx := db.Model(&Course{})
	if category != "" && category != "all" {
		tx = tx.Where("category = ?", category)
	}
	if sort == "hot" {
		tx = tx.Order("view_count desc").Limit(5)
	} else {
		tx = tx.Order("created_at desc")
	}

	tx.Find(&courses)
	c.JSON(200, gin.H{"data": courses})
}

func GetCourseDetailHandler(c *gin.Context) {
	courseID := c.Param("id")
	var course Course
	if err := db.First(&course, courseID).Error; err != nil {
		c.JSON(404, gin.H{"error": "课程不存在"})
		return
	}
	
	// 增加浏览量
	db.Model(&course).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))
	
	// 检查是否已选课（手动解析Token，不强制要求登录）
	isEnrolled := false
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.Contains(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, _ := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) { return []byte(JWT_SECRET), nil })
		if token != nil && token.Valid {
			claims := token.Claims.(jwt.MapClaims)
			uid := uint(claims["user_id"].(float64))
			var count int64
			db.Model(&Enrollment{}).Where("user_id = ? AND course_id = ?", uid, course.ID).Count(&count)
			if count > 0 { isEnrolled = true }
		}
	}

	c.JSON(200, gin.H{"course": course, "is_enrolled": isEnrolled})
}

// --- 需鉴权接口 ---

func UploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil { c.JSON(400, gin.H{"error": "No file"}); return }
	
	bucket := BUCKET_PICTURES
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == ".mp4" || ext == ".avi" { bucket = BUCKET_VIDEOS }
	
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
	src, _ := file.Open(); defer src.Close()
	
	_, err = minioClient.PutObject(context.Background(), bucket, filename, src, file.Size, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		c.JSON(500, gin.H{"error": "上传失败: " + err.Error()})
		return
	}
	
	url := fmt.Sprintf("http://%s/%s/%s", MINIO_ENDPOINT, bucket, filename)
	c.JSON(200, gin.H{"url": url})
}

func CreateCourseHandler(c *gin.Context) {
	var course Course
	if err := c.ShouldBindJSON(&course); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	course.ViewCount = 0
	db.Create(&course)
	c.JSON(200, gin.H{"message": "发布成功"})
}

func UpdateCourseHandler(c *gin.Context) {
	id := c.Param("id")
	userRole := c.MustGet("role").(string)
	userID := c.MustGet("userID").(uint)

	var req Course
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}

	var course Course
	if err := db.First(&course, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "课程不存在"})
		return
	}

	if userRole != "admin" && course.TeacherID != userID {
		c.JSON(403, gin.H{"error": "权限不足"})
		return
	}

	updates := make(map[string]interface{})
	if req.Category != "" { updates["category"] = req.Category }
	if req.Title != "" { updates["title"] = req.Title }
	if req.Description != "" { updates["description"] = req.Description }
	if req.Price >= 0 { updates["price"] = req.Price }
	// 在 UpdateCourseHandler 函数内部的 updates map 赋值部分添加：
	if req.Outline != "" { updates["outline"] = req.Outline }
	if req.HomeworkReq != "" { updates["homework_req"] = req.HomeworkReq }

	db.Model(&course).Updates(updates)
	c.JSON(200, gin.H{"message": "更新成功"})
}

func EnrollHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct { CourseID uint `json:"course_id"` }
	c.ShouldBindJSON(&req)
	
	var count int64
	db.Model(&Enrollment{}).Where("user_id = ? AND course_id = ?", userID, req.CourseID).Count(&count)
	if count > 0 {
		c.JSON(400, gin.H{"error": "已加入该课程"})
		return
	}

	enroll := Enrollment{UserID: userID, CourseID: req.CourseID}
	db.Create(&enroll)
	c.JSON(200, gin.H{"message": "加入成功"})
}

func GetMyCoursesHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var enrolls []Enrollment
	db.Preload("Course").Where("user_id = ?", userID).Find(&enrolls)
	c.JSON(200, gin.H{"data": enrolls})
}

func SubmitHomeworkHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var hw Homework
	c.ShouldBindJSON(&hw)
	hw.StudentID = userID
	
	var exist Homework
	if err := db.Where("course_id = ? AND student_id = ?", hw.CourseID, userID).First(&exist).Error; err == nil {
		exist.Content = hw.Content
		db.Save(&exist)
	} else {
		db.Create(&hw)
	}
	c.JSON(200, gin.H{"message": "提交成功"})
}

func GetHomeworkHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	courseID := c.Query("course_id")
	var hw Homework
	if err := db.Where("course_id = ? AND student_id = ?", courseID, userID).First(&hw).Error; err != nil {
		c.JSON(200, gin.H{"exists": false})
		return
	}
	c.JSON(200, gin.H{"exists": true, "data": hw})
}

func CreateQuestionHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		CourseID uint   `json:"course_id"`
		Content  string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	
	q := Question{
		CourseID:   req.CourseID,
		StudentID:  userID,
		Content:    req.Content,
		IsAnswered: false,
	}
	db.Create(&q)
	c.JSON(200, gin.H{"message": "提问成功"})
}

// 获取某课程的问题列表
func GetCourseQuestionsHandler(c *gin.Context) {
	courseID := c.Query("course_id")
	var questions []Question
	// 预加载学生信息，以便显示是谁问的
	db.Preload("Student").Where("course_id = ?", courseID).Order("created_at desc").Find(&questions)
	c.JSON(200, gin.H{"data": questions})
}

// 教师回复问题
func ReplyQuestionHandler(c *gin.Context) {
	teacherID := c.MustGet("userID").(uint)
	role := c.MustGet("role").(string)
	
	if role != "teacher" && role != "admin" {
		c.JSON(403, gin.H{"error": "权限不足"})
		return
	}

	var req struct {
		ID     uint   `json:"id"`
		Answer string `json:"answer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}

	var q Question
	if err := db.First(&q, req.ID).Error; err != nil {
		c.JSON(404, gin.H{"error": "问题不存在"})
		return
	}

	q.Answer = req.Answer
	q.TeacherID = teacherID
	q.IsAnswered = true
	db.Save(&q)
	c.JSON(200, gin.H{"message": "回复成功"})
}

// --- 教师管理相关 ---

// 教师批改作业
func GradeHomeworkHandler(c *gin.Context) {
	role := c.MustGet("role").(string)
	if role != "teacher" && role != "admin" {
		c.JSON(403, gin.H{"error": "只有教师可以批改"})
		return
	}

	var req struct {
		ID      uint   `json:"id"`
		Score   int    `json:"score"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}

	var hw Homework
	if err := db.First(&hw, req.ID).Error; err != nil {
		c.JSON(404, gin.H{"error": "作业不存在"})
		return
	}

	hw.Score = req.Score
	hw.Comment = req.Comment
	db.Save(&hw)
	c.JSON(200, gin.H{"message": "批改完成"})
}

// 获取教师待办事项 (包括待批改作业和待回复问题)
func GetTeacherDashboardHandler(c *gin.Context) {
	teacherID := c.MustGet("userID").(uint)
	role := c.MustGet("role").(string)
	
	if role != "teacher" {
		c.JSON(403, gin.H{"error": "权限不足"})
		return
	}

	// 1. 查找该老师发布的所有课程ID
	var courseIDs []uint
	db.Model(&Course{}).Where("teacher_id = ?", teacherID).Pluck("id", &courseIDs)

	if len(courseIDs) == 0 {
		c.JSON(200, gin.H{"homeworks": []interface{}{}, "questions": []interface{}{}})
		return
	}

	// 2. 查找这些课程下，分数为0(未批改)的作业
	// 这里我们需要关联 User 表来获取学生名字，但 Homework 结构体定义里没写关联User，
	// 为了简化，我们这里只返回 StudentID，前端如果需要名字，最好在 Homework 结构体加 Student User 关联，
	// 或者前端只显示 ID。为了展示效果，我临时在查询里手动 Join 一下或者简化处理。
	// 这里我们假设前端只显示 StudentID 或者我们修改 Homework 结构体增加 `Student User` 字段（建议方案）。
	// 鉴于不大幅改动已有结构，我们先只返回原始数据。
	
	var homeworks []Homework
	// 简单的逻辑：Score 为 0 视为未批改
	db.Where("course_id IN ? AND score = 0", courseIDs).Find(&homeworks)

	// 3. 查找这些课程下，未回复的问题
	var questions []Question
	db.Preload("Student").Where("course_id IN ? AND is_answered = ?", courseIDs, false).Find(&questions)

	c.JSON(200, gin.H{
		"homeworks": homeworks,
		"questions": questions,
	})
}

// ===========================
// 5. Main 入口
// ===========================

func main() {
	initDB()
	initMinIO()

	r := gin.Default()
	
	// CORS 中间件配置
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api/v1")
	{
		// === 公开路由 (绝对不能加中间件) ===
		api.POST("/register", RegisterHandler)
		api.POST("/login", LoginHandler)
		api.GET("/courses", ListCoursesHandler)      // 课程列表
		api.GET("/courses/:id", GetCourseDetailHandler) // 课程详情

		// === 需登录路由 ===
		auth := api.Group("/")
		auth.Use(AuthMiddleware()) // 只在这里挂载中间件
		{
			auth.POST("/upload", UploadHandler)
			auth.POST("/courses", CreateCourseHandler)
			auth.PUT("/courses/:id", UpdateCourseHandler) // 管理员修改课程
			
			auth.POST("/enroll", EnrollHandler)
			auth.GET("/my-courses", GetMyCoursesHandler)
			auth.POST("/homework", SubmitHomeworkHandler)
			auth.GET("/homework", GetHomeworkHandler)
			auth.POST("/questions", CreateQuestionHandler)      // 学生提问
			auth.GET("/questions", GetCourseQuestionsHandler)   // 获取问题列表
			auth.PUT("/questions/reply", ReplyQuestionHandler)  // 教师回复

			auth.PUT("/homework/grade", GradeHomeworkHandler)       // 教师批改
			auth.GET("/teacher/dashboard", GetTeacherDashboardHandler) // 教师获取待办数据
		}
	}

	r.Run(":8080")
}
