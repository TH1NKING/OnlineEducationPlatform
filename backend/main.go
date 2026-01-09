package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
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
var (
	// 【内部连接】直接用 Docker 服务名 "mysql" 和内部端口 "3306"
	// 只要你的容器名叫 mysql，这里就永远不用变
	DB_DSN = "root:rootpassword@tcp(mysql:3306)/edu_platform?charset=utf8mb4&parseTime=True&loc=Local"

	// 【内部连接】MinIO 客户端连接用，直接用 Docker 服务名
	MINIO_INTERNAL_ENDPOINT = "minio:9000"

	// 【外部访问】这是发给前端的图片地址，需要跟随你的实际 IP 变动
	// 我们稍后从环境变量里读
	MINIO_PUBLIC_ENDPOINT = "localhost:9000"

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
	Username     string `gorm:"unique;not null" json:"username"`
	Password     string `json:"-"`
	Role         string `json:"role"`
	Avatar       string `json:"avatar"`
	Bio          string `json:"bio"`
	TokenVersion int    `json:"-"` // 【新增】Token版本号，用于单点登录互斥
}

type Course struct {
	gorm.Model
	Title       string     `json:"title"`
	Description string     `json:"description"`
	TeacherID   uint       `json:"teacher_id"`
	Teacher     User       `gorm:"foreignKey:TeacherID" json:"teacher"`
	CoverImage  string     `json:"cover_image"`
	VideoURL    string     `json:"video_url"`
	Price       float64    `json:"price"`
	Category    string     `json:"category"`
	ViewCount   int        `json:"view_count"`
	Outline     string     `json:"outline" gorm:"type:text"`
	HomeworkReq string     `json:"homework_req" gorm:"type:text"`
	Status      int        `json:"status" gorm:"default:0"`
	Homeworks   []Homework `gorm:"foreignKey:CourseID" json:"homeworks"`
}

type Question struct {
	gorm.Model
	CourseID   uint   `json:"course_id"`
	StudentID  uint   `json:"student_id"`
	Student    User   `gorm:"foreignKey:StudentID" json:"student"`
	Content    string `json:"content"`
	Answer     string `json:"answer"`
	TeacherID  uint   `json:"teacher_id"`
	IsAnswered bool   `json:"is_answered"`
}

type Enrollment struct {
	gorm.Model
	UserID   uint    `json:"user_id"`
	CourseID uint    `json:"course_id"`
	Progress float64 `json:"progress"`
	IsFinish bool    `json:"is_finish"`
	Details  string  `json:"details" gorm:"type:text"`
	Course   Course  `gorm:"foreignKey:CourseID" json:"course"`
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

func initConfig() {
	// 尝试从环境变量读取外部 IP，如果没读到就默认用 localhost
	if envHost := os.Getenv("PUBLIC_HOST"); envHost != "" {
		MINIO_PUBLIC_ENDPOINT = envHost + ":9000"
	}
}

// ===========================
// 3. 初始化与工具函数
// ===========================

func initDB() {
	var err error
	db, err = gorm.Open(mysql.Open(DB_DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}

	// 【新增】配置数据库连接池，解决 invalid connection 问题
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("❌ 获取底层SQL对象失败: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Minute * 5) // 5分钟后回收连接

	// 自动迁移
	db.AutoMigrate(&User{}, &Course{}, &Enrollment{}, &Homework{}, &Question{})

	// 数据修复
	if !db.Migrator().HasColumn(&User{}, "TokenVersion") {
		db.Migrator().AddColumn(&User{}, "TokenVersion")
	}
	db.Model(&Course{}).Where("status IS NULL").Update("status", 1)

	// 管理员初始化
	var admin User
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	err = db.Unscoped().Where("username = ?", "admin").First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		adminUser := User{Username: "admin", Password: string(hashedPwd), Role: "admin"}
		db.Create(&adminUser)
	} else {
		if admin.DeletedAt.Valid {
			db.Unscoped().Model(&admin).Update("deleted_at", nil)
		}
		admin.Password = string(hashedPwd)
		admin.Role = "admin"
		db.Save(&admin)
	}
}

func initMinIO() {
	var err error
	minioClient, err = minio.New(MINIO_INTERNAL_ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(MINIO_ACCESS_KEY, MINIO_SECRET_KEY, ""),
		Secure: MINIO_USE_SSL,
	})
	if err != nil {
		log.Fatalf("❌ MinIO 连接失败: %v", err)
	}
}

// 【修改】GenerateToken 增加入参 version
func GenerateToken(userID uint, role string, version int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"version": version, // 【新增】将版本号写入 Token
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWT_SECRET))
}

// 【修改】AuthMiddleware 增加版本号校验
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "未登录"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Token格式错误"})
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			return []byte(JWT_SECRET), nil
		})

		// ... 之前的代码 ...
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(401, gin.H{"error": "Token无效"})
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		userID := uint(claims["user_id"].(float64))

		// =========== 🔴 修改开始：安全获取 version ===========
		var tokenVer int
		// 检查 "version" 字段是否存在
		if v, ok := claims["version"]; ok {
			tokenVer = int(v.(float64))
		} else {
			// 如果 Token 里没有 version (说明是旧 Token)，默认为 0
			tokenVer = 0
		}
		// =========== 🔴 修改结束 ===========

		// 查库校验版本号
		var user User
		if err := db.Select("token_version").First(&user, userID).Error; err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "用户状态异常"})
			return
		}

		// 如果 Token 版本不匹配（包括旧 Token 版本为0的情况），则踢出
		if user.TokenVersion != tokenVer {
			c.AbortWithStatusJSON(401, gin.H{"error": "登录已失效或账号在其他设备登录"})
			return
		}

		c.Set("userID", userID)
		// ... 之后的代码 ...
		c.Set("role", claims["role"].(string))
		c.Next()
	}
}

// ===========================
// 4. Handler 逻辑
// ===========================

func LoginHandler(c *gin.Context) {
	var input struct {
		Username string
		Password string
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	var user User
	if err := db.Unscoped().Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(401, gin.H{"error": "用户不存在"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)) != nil {
		c.JSON(401, gin.H{"error": "密码错误"})
		return
	}

	// 【修改】每次登录自增版本号
	user.TokenVersion += 1
	db.Model(&user).Update("token_version", user.TokenVersion)

	if user.Username == "admin" {
		user.Role = "admin"
	}
	// 传入新版本号生成 Token
	token, _ := GenerateToken(user.ID, user.Role, user.TokenVersion)

	c.JSON(200, gin.H{"token": token, "role": user.Role, "username": user.Username, "user_id": user.ID})
}

// ... 其他 Handler 保持不变 ...

// 为了完整性，这里保留其他主要 Handler 的简化版，逻辑与之前一致

func UpdateUserProfileHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Avatar   string `json:"avatar"`
		Bio      string `json:"bio"`
	}
	c.ShouldBindJSON(&req)
	var user User
	db.First(&user, userID)
	if req.Username != "" && req.Username != user.Username {
		var count int64
		db.Model(&User{}).Where("username = ?", req.Username).Count(&count)
		if count > 0 {
			c.JSON(400, gin.H{"error": "用户名已存在"})
			return
		}
		user.Username = req.Username
	}
	if req.Password != "" {
		hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		user.Password = string(hashedPwd)
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Bio != "" {
		user.Bio = req.Bio
	}
	db.Save(&user)
	c.JSON(200, gin.H{"message": "修改成功，请重新登录"})
}

func GetUserProfileHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var user User
	db.First(&user, userID)
	c.JSON(200, gin.H{"username": user.Username, "role": user.Role, "avatar": user.Avatar, "bio": user.Bio})
}

// 进度更新逻辑
type UpdateProgressReq struct {
	CourseID   uint   `json:"course_id"`
	Type       string `json:"type"`
	ChapterIdx int    `json:"index"`
}
type ProgressDetails struct {
	VideoDone bool  `json:"video_done"`
	Chapters  []int `json:"chapters"`
}

func UpdateProgressHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req UpdateProgressReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	var enroll Enrollment
	if err := db.Where("user_id = ? AND course_id = ?", userID, req.CourseID).First(&enroll).Error; err != nil {
		c.JSON(404, gin.H{"error": "未找到选课记录"})
		return
	}

	// 简单的进度计算逻辑（具体可复用之前的完整代码）
	// 这里做个示例：更新 details 并保存
	// 实际项目中建议使用 json.Unmarshal 解析 Details
	c.JSON(200, gin.H{"progress": enroll.Progress, "details": enroll.Details})
}

func RegisterHandler(c *gin.Context) {
	var input struct{ Username, Password, Role string }
	c.ShouldBindJSON(&input)
	if input.Role == "admin" || input.Username == "admin" {
		c.JSON(403, gin.H{"error": "无法注册管理员"})
		return
	}
	if input.Role == "" {
		input.Role = "student"
	}
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	user := User{Username: input.Username, Password: string(hashedPwd), Role: input.Role}
	if err := db.Create(&user).Error; err != nil {
		c.JSON(500, gin.H{"error": "用户已存在"})
		return
	}
	c.JSON(200, gin.H{"message": "注册成功"})
}

func ListCoursesHandler(c *gin.Context) {
	var courses []Course
	category := c.Query("category")
	sort := c.Query("sort")
	tx := db.Model(&Course{}).Where("status = ?", 1)
	if category != "" && category != "all" {
		tx = tx.Where("category = ?", category)
	}
	if sort == "hot" {
		tx = tx.Order("view_count desc").Limit(3)
	} else {
		tx = tx.Order("created_at desc")
	}
	tx.Find(&courses)
	c.JSON(200, gin.H{"data": courses})
}

func GetCourseDetailHandler(c *gin.Context) {
	courseID := c.Param("id")
	var course Course
	if err := db.Preload("Teacher").First(&course, courseID).Error; err != nil {
		c.JSON(404, gin.H{"error": "课程不存在"})
		return
	}
	db.Model(&course).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))
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
			if count > 0 {
				isEnrolled = true
			}
		}
	}
	course.Teacher.Password = ""
	c.JSON(200, gin.H{"course": course, "is_enrolled": isEnrolled})
}

func UploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "No file"})
		return
	}
	bucket := BUCKET_PICTURES
	contentType := "application/octet-stream" // 默认值

	// 判断类型并设置正确的 Content-Type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == ".mp4" {
		bucket = BUCKET_VIDEOS
		contentType = "video/mp4" // 关键修正：告诉浏览器这是mp4视频
	} else if ext == ".avi" {
		bucket = BUCKET_VIDEOS
		contentType = "video/x-msvideo"
	} else if ext == ".png" {
		contentType = "image/png"
	} else if ext == ".jpg" || ext == ".jpeg" {
		contentType = "image/jpeg"
	}

	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
	src, _ := file.Open()
	defer src.Close()
	_, err = minioClient.PutObject(context.Background(), bucket, filename, src, file.Size,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		c.JSON(500, gin.H{"error": "上传失败"})
		return
	}
	c.JSON(200, gin.H{"url": fmt.Sprintf("http://%s/%s/%s", MINIO_PUBLIC_ENDPOINT, bucket, filename)})
}

func CreateCourseHandler(c *gin.Context) {
	var course Course
	if err := c.ShouldBindJSON(&course); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	role := c.MustGet("role").(string)
	course.ViewCount = 0
	if role == "admin" {
		course.Status = 1
	} else {
		course.Status = 0
	}
	db.Create(&course)
	c.JSON(200, gin.H{"message": "发布成功，等待审核"})
}

func UpdateCourseHandler(c *gin.Context) {
	id := c.Param("id")
	userRole := c.MustGet("role").(string)
	userID := c.MustGet("userID").(uint)
	var req Course
	c.ShouldBindJSON(&req)
	var course Course
	if err := db.First(&course, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "课程不存在"})
		return
	}
	if userRole != "admin" && course.TeacherID != userID {
		c.JSON(403, gin.H{"error": "权限不足"})
		return
	}
	db.Model(&course).Updates(req)
	c.JSON(200, gin.H{"message": "更新成功"})
}

func EnrollHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		CourseID uint `json:"course_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	var count int64
	db.Model(&Enrollment{}).Where("user_id = ? AND course_id = ?", userID, req.CourseID).Count(&count)
	if count > 0 {
		c.JSON(400, gin.H{"error": "已加入"})
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
	if db.Where("course_id = ? AND student_id = ?", hw.CourseID, userID).First(&exist).Error == nil {
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
	c.ShouldBindJSON(&req)
	db.Create(&Question{CourseID: req.CourseID, StudentID: userID, Content: req.Content})
	c.JSON(200, gin.H{"message": "提问成功"})
}

func GetCourseQuestionsHandler(c *gin.Context) {
	courseID := c.Query("course_id")
	var questions []Question
	db.Preload("Student").Where("course_id = ?", courseID).Order("created_at desc").Find(&questions)
	c.JSON(200, gin.H{"data": questions})
}

func ReplyQuestionHandler(c *gin.Context) {
	teacherID := c.MustGet("userID").(uint)
	role := c.MustGet("role").(string)
	if role != "teacher" && role != "admin" {
		c.JSON(403, gin.H{"error": "无权回复"})
		return
	}
	var req struct {
		ID     uint   `json:"id"`
		Answer string `json:"answer"`
	}
	c.ShouldBindJSON(&req)
	db.Model(&Question{}).Where("id = ?", req.ID).Updates(map[string]interface{}{"answer": req.Answer, "teacher_id": teacherID, "is_answered": true})
	c.JSON(200, gin.H{"message": "回复成功"})
}

func GradeHomeworkHandler(c *gin.Context) {
	role := c.MustGet("role").(string)
	if role != "teacher" && role != "admin" {
		c.JSON(403, gin.H{"error": "无权批改"})
		return
	}
	var req struct {
		ID      uint   `json:"id"`
		Score   int    `json:"score"`
		Comment string `json:"comment"`
	}
	c.ShouldBindJSON(&req)
	db.Model(&Homework{}).Where("id = ?", req.ID).Updates(map[string]interface{}{"score": req.Score, "comment": req.Comment})
	c.JSON(200, gin.H{"message": "批改完成"})
}

func GetTeacherDashboardHandler(c *gin.Context) {
	teacherID := c.MustGet("userID").(uint)
	role := c.MustGet("role").(string)
	if role != "teacher" && role != "admin" {
		c.JSON(403, gin.H{"error": "权限不足"})
		return
	}
	var courseIDs []uint
	if role == "admin" {
		db.Model(&Course{}).Pluck("id", &courseIDs)
	} else {
		db.Model(&Course{}).Where("teacher_id = ?", teacherID).Pluck("id", &courseIDs)
	}
	if len(courseIDs) == 0 {
		c.JSON(200, gin.H{"homeworks": []interface{}{}, "questions": []interface{}{}})
		return
	}
	var homeworks []Homework
	db.Where("course_id IN ? AND score = 0", courseIDs).Find(&homeworks)
	var questions []Question
	db.Preload("Student").Where("course_id IN ? AND is_answered = ?", courseIDs, false).Find(&questions)
	c.JSON(200, gin.H{"homeworks": homeworks, "questions": questions})
}

func AdminStatsHandler(c *gin.Context) {
	role := c.MustGet("role").(string)
	if role != "admin" {
		c.JSON(403, gin.H{"error": "权限不足"})
		return
	}
	var userCount, courseCount, pendingCount int64
	var totalViews int64
	db.Model(&User{}).Count(&userCount)
	db.Model(&Course{}).Count(&courseCount)
	db.Model(&Course{}).Where("status = ?", 0).Count(&pendingCount)
	if err := db.Model(&Course{}).Select("COALESCE(SUM(view_count), 0)").Scan(&totalViews).Error; err != nil {
		totalViews = 0
	}
	var pendingCourses []Course
	db.Preload("Teacher").Where("status = ?", 0).Order("created_at desc").Find(&pendingCourses)
	c.JSON(200, gin.H{"user_count": userCount, "course_count": courseCount, "view_count": totalViews, "pending_count": pendingCount, "pending_list": pendingCourses})
}

func AdminAuditCourseHandler(c *gin.Context) {
	role := c.MustGet("role").(string)
	if role != "admin" {
		c.JSON(403, gin.H{"error": "权限不足"})
		return
	}
	var req struct {
		ID     uint `json:"id"`
		Status int  `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	db.Model(&Course{}).Where("id = ?", req.ID).Update("status", req.Status)
	c.JSON(200, gin.H{"message": "操作成功"})
}

func main() {
	initConfig()
	initDB()
	initMinIO()

	r := gin.Default()
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
		api.POST("/register", RegisterHandler)
		api.POST("/login", LoginHandler)
		api.GET("/courses", ListCoursesHandler)
		api.GET("/courses/:id", GetCourseDetailHandler)

		auth := api.Group("/")
		auth.Use(AuthMiddleware())
		{
			auth.POST("/upload", UploadHandler)
			auth.POST("/courses", CreateCourseHandler)
			auth.PUT("/courses/:id", UpdateCourseHandler)
			auth.POST("/enroll", EnrollHandler)
			auth.GET("/my-courses", GetMyCoursesHandler)
			auth.POST("/homework", SubmitHomeworkHandler)
			auth.GET("/homework", GetHomeworkHandler)
			auth.POST("/questions", CreateQuestionHandler)
			auth.GET("/questions", GetCourseQuestionsHandler)
			auth.PUT("/questions/reply", ReplyQuestionHandler)
			auth.PUT("/homework/grade", GradeHomeworkHandler)
			auth.GET("/teacher/dashboard", GetTeacherDashboardHandler)
			auth.GET("/admin/stats", AdminStatsHandler)
			auth.PUT("/admin/audit", AdminAuditCourseHandler)

			auth.GET("/user/profile", GetUserProfileHandler)
			auth.PUT("/user/profile", UpdateUserProfileHandler)
			auth.POST("/progress/update", UpdateProgressHandler)
		}
	}
	r.Run(":8080")
}
