package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Модели данных
type User struct {
	ID         uint  `gorm:"primaryKey"`
	GitHubID   int64 `gorm:"unique"`
	TelegramID int64 `gorm:"unique"`
	FullName   string
	GroupName  string
	Roles      []Role `gorm:"many2many:user_roles;"`
}

type Role struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"unique"`
}

type UserRequest struct {
	GitHubID   int64    `json:"github_id"`
	TelegramID int64    `json:"telegram_id"`
	FullName   string   `json:"full_name"`
	GroupName  string   `json:"group_name"`
	Roles      []string `json:"roles"`
}

var db *gorm.DB
var apiKey string

func main() {
	// Загрузка .env файла
	godotenv.Load()

	// Инициализация БД
	initDB()

	// Автомиграция
	db.AutoMigrate(&User{}, &Role{})

	roles := []string{"student", "teacher", "admin"}
	for _, roleName := range roles {
		var role Role
		db.FirstOrCreate(&role, Role{Name: roleName})
	}

	// Инициализация роутера
	r := gin.Default()
	apiKey = os.Getenv("API_KEY")

	// Группа API с проверкой ключа
	api := r.Group("/api", authMiddleware)
	{
		api.POST("/register", registerUser)
		api.DELETE("/user/:id", deleteUser)
		api.PUT("/user/:id/roles", updateUserRoles)
		api.PUT("/user/:id", updateUserInfo)
		api.GET("/user/:id", getUser)
		api.GET("/user/:id/roles", getUserRoles)
	}

	// Запуск сервера
	r.Run(":8080")
}

func initDB() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database")
	}
}

func authMiddleware(c *gin.Context) {
	providedKey := c.GetHeader("X-API-Key")
	if providedKey != apiKey {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		return
	}
	c.Next()
}

// Обработчики API
func registerUser(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var count int64
	db.Model(&User{}).Where("github_id = ? OR telegram_id = ?", req.GitHubID, req.TelegramID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}
	user := User{
		GitHubID:   req.GitHubID,
		TelegramID: req.TelegramID,
		FullName:   req.FullName,
		GroupName:  req.GroupName,
	}

	// Добавляем роль студента по умолчанию
	var studentRole Role
	db.Where("name = ?", "student").First(&studentRole)
	user.Roles = append(user.Roles, studentRole)

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func deleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := db.Delete(&User{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func updateUserRoles(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct{ Roles []string }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	if err := db.Preload("Roles").First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Находим новые роли
	var newRoles []Role
	for _, roleName := range req.Roles {
		var role Role
		db.Where("name = ?", roleName).First(&role)
		newRoles = append(newRoles, role)
	}

	db.Model(&user).Association("Roles").Replace(newRoles)
	c.JSON(http.StatusOK, user)
}

func updateUserInfo(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	db.Model(&user).Updates(User{
		FullName:  req.FullName,
		GroupName: req.GroupName,
	})

	c.JSON(http.StatusOK, user)
}

func getUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user User
	if err := db.Preload("Roles").First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func getUserRoles(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user User
	if err := db.Preload("Roles").First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var roles []string
	for _, role := range user.Roles {
		roles = append(roles, role.Name)
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}
