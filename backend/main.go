package main

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	"encoding/json"
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
	Title       string     `json:"title"`
	Description string     `json:"description"`
	TeacherID   uint       `json:"teacher_id"`
	Teacher     User       `gorm:"foreignKey:TeacherID" json:"teacher"` // 关联教师信息
	CoverImage  string     `json:"cover_image"`
	VideoURL    string     `json:"video_url"`
	Price       float64    `json:"price"`
	Category    string     `json:"category"`
	ViewCount   int        `json:"view_count"`
	Outline     string     `json:"outline" gorm:"type:text"`
	HomeworkReq string     `json:"homework_req" gorm:"type:text"`
	Status      int        `json:"status" gorm:"default:0"` // 0:待审核, 1:已发布, 2:已驳回
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

// 用于数据库存储的 JSON 结构
type ProgressState struct {
	VideoDone bool  `json:"video_done"` // 视频是否看完
	Chapters  []int `json:"chapters"`   // 已完成章节的下标数组 (0, 1, 2...)
}

// 前端请求参数
type UpdateProgressReq struct {
	CourseID   uint   `json:"course_id"`
	Type       string `json:"type"`  // "video" 或 "chapter"
	ChapterIdx int    `json:"index"` // 如果是 chapter，传入章节下标
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

	// --- 数据清洗：将旧数据的Status设为1(已发布)，避免旧课程消失 ---
	db.Model(&Course{}).Where("status IS NULL").Update("status", 1)

	// --- 管理员初始化 ---
	var admin User
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	err = db.Unscoped().Where("username = ?", "admin").First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		adminUser := User{Username: "admin", Password: string(hashedPwd), Role: "admin"}
		db.Create(&adminUser)
		log.Println("✅ 管理员创建成功 -> 账号: admin / 密码: 123456")
	} else {
		if admin.DeletedAt.Valid { db.Unscoped().Model(&admin).Update("deleted_at", nil) }
		admin.Password = string(hashedPwd)
		admin.Role = "admin"
		db.Save(&admin)
		log.Println("✅ 管理员修复成功")
	}
}

func initMinIO() {
	var err error
	minioClient, err = minio.New(MINIO_ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(MINIO_ACCESS_KEY, MINIO_SECRET_KEY, ""),
		Secure: MINIO_USE_SSL,
	})
	if err != nil { log.Fatalf("❌ MinIO 连接失败: %v", err) }
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
		if authHeader == "" { c.AbortWithStatusJSON(401, gin.H{"error": "未登录"}); return }
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" { c.AbortWithStatusJSON(401, gin.H{"error": "Token格式错误"}); return }
		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) { return []byte(JWT_SECRET), nil })
		if err != nil || !token.Valid { c.AbortWithStatusJSON(401, gin.H{"error": "Token无效"}); return }
		claims := token.Claims.(jwt.MapClaims)
		c.Set("userID", uint(claims["user_id"].(float64)))
		c.Set("role", claims["role"].(string))
		c.Next()
	}
}

// ===========================
// 4. Handler 逻辑
// ===========================

func RegisterHandler(c *gin.Context) {
	var input struct { Username string; Password string; Role string }
	if err := c.ShouldBindJSON(&input); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	if input.Role == "admin" || input.Username == "admin" { c.JSON(403, gin.H{"error": "无法注册管理员"}); return }
	if input.Role == "" { input.Role = "student" }
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	user := User{Username: input.Username, Password: string(hashedPwd), Role: input.Role}
	if err := db.Create(&user).Error; err != nil { c.JSON(500, gin.H{"error": "用户已存在"}); return }
	c.JSON(200, gin.H{"message": "注册成功"})
}

func LoginHandler(c *gin.Context) {
	var input struct { Username string; Password string }
	if err := c.ShouldBindJSON(&input); err != nil { c.JSON(400, gin.H{"error": "参数错误"}); return }
	var user User
	if err := db.Unscoped().Where("username = ?", input.Username).First(&user).Error; err != nil { c.JSON(401, gin.H{"error": "用户不存在"}); return }
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)) != nil { c.JSON(401, gin.H{"error": "密码错误"}); return }
	if user.Username == "admin" { user.Role = "admin" }
	token, _ := GenerateToken(user.ID, user.Role)
	c.JSON(200, gin.H{"token": token, "role": user.Role, "username": user.Username, "user_id": user.ID})
}

// --- 公开接口 ---

func ListCoursesHandler(c *gin.Context) {
	var courses []Course
	category := c.Query("category")
	sort := c.Query("sort")

	tx := db.Model(&Course{})
	// 关键修改：只显示已发布(1)的课程
	tx = tx.Where("status = ?", 1)

	if category != "" && category != "all" {
		tx = tx.Where("category = ?", category)
	}

	// === 修改位置 ===
	if sort == "hot" {
		// 原代码是 Limit(5)，修改为 Limit(3) 以仅显示前3名
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
	if err := db.First(&course, courseID).Error; err != nil { c.JSON(404, gin.H{"error": "课程不存在"}); return }
	// 浏览量增加
	db.Model(&course).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))
	
	isEnrolled := false
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.Contains(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, _ := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) { return []byte(JWT_SECRET), nil })
		if token != nil && token.Valid {
			claims := token.Claims.(jwt.MapClaims)
			var count int64
			db.Model(&Enrollment{}).Where("user_id = ? AND course_id = ?", uint(claims["user_id"].(float64)), course.ID).Count(&count)
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
	if ext := strings.ToLower(filepath.Ext(file.Filename)); ext == ".mp4" || ext == ".avi" { bucket = BUCKET_VIDEOS }
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
	src, _ := file.Open(); defer src.Close()
	_, err = minioClient.PutObject(context.Background(), bucket, filename, src, file.Size, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil { c.JSON(500, gin.H{"error": "上传失败"}); return }
	c.JSON(200, gin.H{"url": fmt.Sprintf("http://%s/%s/%s", MINIO_ENDPOINT, bucket, filename)})
}

func CreateCourseHandler(c *gin.Context) {
	var course Course
	if err := c.ShouldBindJSON(&course); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	
	role := c.MustGet("role").(string)
	
	course.ViewCount = 0
	// 关键修改：教师创建默认为0(待审核)，管理员创建直接发布
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
	if err := db.First(&course, id).Error; err != nil { c.JSON(404, gin.H{"error": "课程不存在"}); return }
	if userRole != "admin" && course.TeacherID != userID { c.JSON(403, gin.H{"error": "权限不足"}); return }
	
	// 更新逻辑...
	db.Model(&course).Updates(req) // 简化写法，实际项目需指定字段
	c.JSON(200, gin.H{"message": "更新成功"})
}

// --- 管理员特有接口 ---

// 获取系统监控统计
func AdminStatsHandler(c *gin.Context) {
	role := c.MustGet("role").(string)
	if role != "admin" {
		c.JSON(403, gin.H{"error": "权限不足"})
		return
	}

	var userCount, courseCount, pendingCount int64
	var totalViews int64

	// 统计基础数据
	db.Model(&User{}).Count(&userCount)
	db.Model(&Course{}).Count(&courseCount)
	db.Model(&Course{}).Where("status = ?", 0).Count(&pendingCount)

	// 修复：使用 COALESCE 处理 sum 为 NULL 的情况，防止程序崩溃
	// 如果没有记录，SQL sum 返回 null，导致 scan 失败。COALESCE(..., 0) 强制转为 0
	// COALESCE(..., 0) 保证了即使没有数据，数据库也会返回 0，而不是 NULL
	if err := db.Model(&Course{}).Select("COALESCE(SUM(view_count), 0)").Scan(&totalViews).Error; err != nil {
		// 即使出错也给个默认值，防止接口挂掉
		totalViews = 0
		log.Println("统计浏览量异常:", err)
	}

	// 获取待审核课程列表
	var pendingCourses []Course
	// 预加载 Teacher 信息以便前端显示是谁提交的
	db.Preload("Teacher").Where("status = ?", 0).Order("created_at desc").Find(&pendingCourses)

	c.JSON(200, gin.H{
		"user_count":    userCount,
		"course_count":  courseCount,
		"view_count":    totalViews,
		"pending_count": pendingCount,
		"pending_list":  pendingCourses,
	})
}

// 审核课程
func AdminAuditCourseHandler(c *gin.Context) {
	role := c.MustGet("role").(string)
	if role != "admin" {
		c.JSON(403, gin.H{"error": "权限不足"})
		return
	}

	var req struct {
		ID     uint `json:"id"`
		Status int  `json:"status"` // 1:通过, 2:驳回
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}

	// 更新状态
	if err := db.Model(&Course{}).Where("id = ?", req.ID).Update("status", req.Status).Error; err != nil {
		c.JSON(500, gin.H{"error": "数据库更新失败"}); return
	}
	c.JSON(200, gin.H{"message": "操作成功"})
}
// --- 其他原有接口 (略微简化保留) ---
func EnrollHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		CourseID uint `json:"course_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}

	// 检查是否已存在
	var count int64
	db.Model(&Enrollment{}).Where("user_id = ? AND course_id = ?", userID, req.CourseID).Count(&count)
	if count > 0 {
		c.JSON(400, gin.H{"error": "已加入该课程"})
		return
	}

	// === 修复：增加错误检查 ===
	enroll := Enrollment{UserID: userID, CourseID: req.CourseID}
	if err := db.Create(&enroll).Error; err != nil {
		log.Println("❌ 加入课程失败:", err) // 打印具体错误到控制台
		c.JSON(500, gin.H{"error": "加入课程失败，请联系管理员"})
		return
	}
	
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
		exist.Content = hw.Content; db.Save(&exist)
	} else { db.Create(&hw) }
	c.JSON(200, gin.H{"message": "提交成功"})
}

func GetHomeworkHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint); courseID := c.Query("course_id")
	var hw Homework
	if err := db.Where("course_id = ? AND student_id = ?", courseID, userID).First(&hw).Error; err != nil { c.JSON(200, gin.H{"exists": false}); return }
	c.JSON(200, gin.H{"exists": true, "data": hw})
}

func CreateQuestionHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct{ CourseID uint `json:"course_id"`; Content string `json:"content"` }
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
	teacherID := c.MustGet("userID").(uint); role := c.MustGet("role").(string)
	if role != "teacher" && role != "admin" { c.JSON(403, gin.H{"error": "无权回复"}); return }
	var req struct{ ID uint `json:"id"`; Answer string `json:"answer"` }
	c.ShouldBindJSON(&req)
	db.Model(&Question{}).Where("id = ?", req.ID).Updates(map[string]interface{}{"answer": req.Answer, "teacher_id": teacherID, "is_answered": true})
	c.JSON(200, gin.H{"message": "回复成功"})
}

func GradeHomeworkHandler(c *gin.Context) {
	role := c.MustGet("role").(string)
	if role != "teacher" && role != "admin" { c.JSON(403, gin.H{"error": "无权批改"}); return }
	var req struct{ ID uint `json:"id"`; Score int `json:"score"`; Comment string `json:"comment"` }
	c.ShouldBindJSON(&req)
	db.Model(&Homework{}).Where("id = ?", req.ID).Updates(map[string]interface{}{"score": req.Score, "comment": req.Comment})
	c.JSON(200, gin.H{"message": "批改完成"})
}

func GetTeacherDashboardHandler(c *gin.Context) {
	teacherID := c.MustGet("userID").(uint); role := c.MustGet("role").(string)
	if role != "teacher" && role != "admin" { c.JSON(403, gin.H{"error": "权限不足"}); return }
	var courseIDs []uint
	if role == "admin" { db.Model(&Course{}).Pluck("id", &courseIDs) } else { db.Model(&Course{}).Where("teacher_id = ?", teacherID).Pluck("id", &courseIDs) }
	if len(courseIDs) == 0 { c.JSON(200, gin.H{"homeworks": []interface{}{}, "questions": []interface{}{}}); return }
	var homeworks []Homework; db.Where("course_id IN ? AND score = 0", courseIDs).Find(&homeworks)
	var questions []Question; db.Preload("Student").Where("course_id IN ? AND is_answered = ?", courseIDs, false).Find(&questions)
	c.JSON(200, gin.H{"homeworks": homeworks, "questions": questions})
}



func UpdateProgressHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req UpdateProgressReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}

	// 1. 获取选课记录
	var enroll Enrollment
	if err := db.Where("user_id = ? AND course_id = ?", userID, req.CourseID).First(&enroll).Error; err != nil {
		c.JSON(404, gin.H{"error": "未找到选课记录"})
		return
	}

	// 2. 解析当前的进度详情
	var state ProgressState
	if enroll.Details != "" {
		json.Unmarshal([]byte(enroll.Details), &state)
	}
	if state.Chapters == nil {
		state.Chapters = []int{}
	}

	// 3. 更新状态
	if req.Type == "video" {
		state.VideoDone = true
	} else if req.Type == "chapter" {
		// 检查是否已经存在，避免重复添加
		exists := false
		for _, idx := range state.Chapters {
			if idx == req.ChapterIdx {
				exists = true
				break
			}
		}
		if !exists {
			state.Chapters = append(state.Chapters, req.ChapterIdx)
		}
	}

	// 4. 获取课程大纲以计算总章节数
	var course Course
	db.First(&course, req.CourseID)
	
	// 解析大纲计算章节总数
	var outline []map[string]interface{}
	totalChapters := 0
	if course.Outline != "" {
		json.Unmarshal([]byte(course.Outline), &outline)
		totalChapters = len(outline)
	}

	// 5. 核心算法：计算进度
	// 视频占 50%
	// 章节占 50% (平均分配给每个章节)
	newProgress := 0.0

	// 视频部分
	if state.VideoDone {
		newProgress += 50.0
	}

	// 章节部分
	if totalChapters > 0 {
		chapterWeight := 50.0 / float64(totalChapters)
		newProgress += float64(len(state.Chapters)) * chapterWeight
	} else {
		// 如果没有章节，视频看完就算100%
		if state.VideoDone { newProgress = 100.0 }
	}

	// 封顶 100%
	if newProgress > 100.0 {
		newProgress = 100.0
	}

	// 6. 保存回数据库
	stateBytes, _ := json.Marshal(state)
	enroll.Details = string(stateBytes)
	enroll.Progress = newProgress
	enroll.IsFinish = (newProgress >= 100.0)

	db.Save(&enroll)

	c.JSON(200, gin.H{
		"message":  "进度更新成功",
		"progress": newProgress,
		"details":  state, // 返回最新详情给前端更新UI
	})
}

func main() {
	initDB()
	initMinIO() // 确保 MinIO 连接

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
		api.GET("/courses", ListCoursesHandler)         // 仅显示 status=1 的课程
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
			auth.POST("/progress/update", UpdateProgressHandler) // 新增这一行

			// === 管理员路由 ===
			// 确保这里的路径与前端请求完全一致
			auth.GET("/admin/stats", AdminStatsHandler)
			auth.PUT("/admin/audit", AdminAuditCourseHandler)
		}
	}

	r.Run(":8080")
}