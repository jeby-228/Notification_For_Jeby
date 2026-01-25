package main

import (
	"context"
	"log"
	"time"

	"member_API/auth"
	"member_API/config"
	"member_API/controllers"
	_ "member_API/docs"
	"member_API/models"
	"member_API/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// @title Member API
// @version 1.0
// @description 這是一個使用 Go、Gin 框架和 PostgreSQL 構建的 RESTful 和 GraphQL API 服務，提供會員管理功能和 JWT 認證
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:9876
// @BasePath /api/v1
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT 認證，格式：Bearer {token}

var db *gorm.DB

func initPostgreSQL(cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if cfg.Database.DSN == "" {
		log.Println("Warning: POSTGRES_DSN not set")
		return nil
	}

	gormDB, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}

	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	if err := sqlDB.PingContext(ctx); err != nil {
		return err
	}

	if err := gormDB.WithContext(ctx).AutoMigrate(
		&models.Tenants{},
		&models.Member{},
		&models.NotificationProvider{},
		&models.MemberNotificationPreference{},
		&models.NotificationLog{},
		&models.MemberDeviceToken{},
	); err != nil {
		return err
	}

	db = gormDB
	controllers.SetupUserController(db)
	controllers.SetupNotificationController(db)
	auth.SetAPIKeyMiddlewareDB(db)

	log.Println("Connected to PostgreSQL")
	return nil
}

// HealthCheck 健康檢查端點
// @Summary 健康檢查
// @Description 檢查服務器狀態和數據庫連接狀態
// @Tags 系統
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "服務正常"
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	status := "OK"
	dbStatus := "Connected"
	if db == nil {
		dbStatus = "Disconnected"
	} else if sqlDB, err := db.DB(); err != nil {
		dbStatus = "Error: " + err.Error()
	} else if err := sqlDB.PingContext(context.Background()); err != nil {
		dbStatus = "Error: " + err.Error()
	}
	c.JSON(200, gin.H{
		"status":          status,
		"postgres_status": dbStatus,
	})
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	cfg := config.Load()

	if err := initPostgreSQL(cfg); err != nil {
		log.Printf("Warning: PostgreSQL failed: %v\n", err)
		log.Println("Starting without database...")
	} else {
		defer func() {
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}()
	}

	router := gin.Default()
	routes.SetupRouter(router)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/health", HealthCheck)

	log.Println("Server starting on :" + cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal(err)
	}
}
