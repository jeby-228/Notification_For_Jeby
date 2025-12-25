package controllers

import (
	"net/http"
	"strconv"

	"member_API/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var productDB *gorm.DB

// SetupProductController stores the shared database handle for product controller use.
func SetupProductController(database *gorm.DB) {
	productDB = database
}

// ProductResponse represents a simplified product record for API responses.
type ProductResponse struct {
	ID                 uint    `json:"id" example:"1"`
	ProductName        string  `json:"product_name" example:"iPhone 15 Pro"`
	ProductPrice       float64 `json:"product_price" example:"35900"`
	ProductDescription string  `json:"product_description" example:"最新款 iPhone"`
	ProductImage       string  `json:"product_image" example:"https://example.com/image.jpg"`
	ProductStock       int     `json:"product_stock" example:"100"`
}

// CreateProductRequest represents the request body for creating a product.
type CreateProductRequest struct {
	ProductName        string  `json:"product_name" binding:"required" example:"iPhone 15 Pro"`
	ProductPrice       float64 `json:"product_price" binding:"required,gt=0" example:"35900"`
	ProductDescription string  `json:"product_description" example:"最新款 iPhone"`
	ProductImage       string  `json:"product_image" example:"https://example.com/image.jpg"`
	ProductStock       int     `json:"product_stock" binding:"required,gte=0" example:"100"`
}

// UpdateProductRequest represents the request body for updating a product.
type UpdateProductRequest struct {
	ProductName        *string  `json:"product_name" example:"iPhone 15 Pro Max"`
	ProductPrice       *float64 `json:"product_price" example:"42900"`
	ProductDescription *string  `json:"product_description" example:"更新的描述"`
	ProductImage       *string  `json:"product_image" example:"https://example.com/new-image.jpg"`
	ProductStock       *int     `json:"product_stock" example:"50"`
}

// GetProducts returns a collection of products from the database.
// @Summary 獲取所有產品
// @Description 獲取產品列表，最多返回 100 條記錄，需要 JWT 認證
// @Tags 產品
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "限制返回數量" default(50) minimum(1) maximum(100)
// @Param offset query int false "偏移量" default(0) minimum(0)
// @Success 200 {object} map[string]interface{} "獲取成功"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /products [get]
func GetProducts(c *gin.Context) {
	if productDB == nil {
		c.JSON(http.StatusOK, gin.H{
			"products": []ProductResponse{},
			"message":  "database connection not configured",
		})
		return
	}

	// 解析分頁參數
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}

	// 使用 Service 層
	svc := services.NewProductService(productDB)
	products, total, err := svc.GetProducts(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	productResponses := make([]ProductResponse, len(products))
	for i, product := range products {
		productResponses[i] = ProductResponse{
			ID:                 product.ID,
			ProductName:        product.ProductName,
			ProductPrice:       product.ProductPrice,
			ProductDescription: product.ProductDescription,
			ProductImage:       product.ProductImage,
			ProductStock:       product.ProductStock,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"products": productResponses,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetProductByID returns a single product by ID from the database.
// @Summary 根據 ID 獲取產品
// @Description 根據產品 ID 獲取單個產品的詳細信息，需要 JWT 認證
// @Tags 產品
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "產品 ID" example(1)
// @Success 200 {object} map[string]ProductResponse "獲取成功"
// @Failure 400 {object} map[string]string "無效的產品 ID"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 404 {object} map[string]string "產品不存在"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /product/{id} [get]
func GetProductByID(c *gin.Context) {
	if productDB == nil {
		c.JSON(http.StatusOK, gin.H{
			"product": nil,
			"message": "database connection not configured",
		})
		return
	}

	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	// 使用 Service 層
	svc := services.NewProductService(productDB)
	product, err := svc.GetProductByID(uint(productID))
	if err != nil {
		if err.Error() == "產品不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"product": ProductResponse{
			ID:                 product.ID,
			ProductName:        product.ProductName,
			ProductPrice:       product.ProductPrice,
			ProductDescription: product.ProductDescription,
			ProductImage:       product.ProductImage,
			ProductStock:       product.ProductStock,
		},
	})
}

// CreateProduct creates a new product in the database.
// @Summary 創建產品
// @Description 創建新產品，需要 JWT 認證
// @Tags 產品
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param product body CreateProductRequest true "產品信息"
// @Success 201 {object} map[string]ProductResponse "創建成功"
// @Failure 400 {object} map[string]string "請求參數錯誤"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /product [post]
func CreateProduct(c *gin.Context) {
	if productDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection not configured"})
		return
	}

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 獲取當前用戶 ID（從 JWT token 中）
	userID, exists := c.Get("user_id")
	var creatorID uint
	if exists {
		if id, ok := userID.(uint); ok {
			creatorID = id
		}
	}

	// 使用 Service 層
	svc := services.NewProductService(productDB)
	product, err := svc.CreateProduct(
		req.ProductName,
		req.ProductPrice,
		req.ProductDescription,
		req.ProductImage,
		req.ProductStock,
		creatorID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"product": ProductResponse{
			ID:                 product.ID,
			ProductName:        product.ProductName,
			ProductPrice:       product.ProductPrice,
			ProductDescription: product.ProductDescription,
			ProductImage:       product.ProductImage,
			ProductStock:       product.ProductStock,
		},
		"message": "product created successfully",
	})
}

// UpdateProduct updates an existing product in the database.
// @Summary 更新產品
// @Description 根據產品 ID 更新產品信息，需要 JWT 認證
// @Tags 產品
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "產品 ID" example(1)
// @Param product body UpdateProductRequest true "要更新的產品信息"
// @Success 200 {object} map[string]ProductResponse "更新成功"
// @Failure 400 {object} map[string]string "請求參數錯誤"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 404 {object} map[string]string "產品不存在"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /product/{id} [put]
func UpdateProduct(c *gin.Context) {
	if productDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection not configured"})
		return
	}

	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 獲取當前用戶 ID
	userID, exists := c.Get("user_id")
	var modifierID uint
	if exists {
		if id, ok := userID.(uint); ok {
			modifierID = id
		}
	}

	// 構建更新欄位
	updates := make(map[string]interface{})
	if req.ProductName != nil {
		updates["product_name"] = *req.ProductName
	}
	if req.ProductPrice != nil {
		updates["product_price"] = *req.ProductPrice
	}
	if req.ProductDescription != nil {
		updates["product_description"] = *req.ProductDescription
	}
	if req.ProductImage != nil {
		updates["product_image"] = *req.ProductImage
	}
	if req.ProductStock != nil {
		updates["product_stock"] = *req.ProductStock
	}

	// 使用 Service 層
	svc := services.NewProductService(productDB)
	product, err := svc.UpdateProduct(uint(productID), updates, modifierID)
	if err != nil {
		if err.Error() == "產品不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"product": ProductResponse{
			ID:                 product.ID,
			ProductName:        product.ProductName,
			ProductPrice:       product.ProductPrice,
			ProductDescription: product.ProductDescription,
			ProductImage:       product.ProductImage,
			ProductStock:       product.ProductStock,
		},
		"message": "product updated successfully",
	})
}

// DeleteProduct soft deletes a product by ID from the database.
// @Summary 刪除產品
// @Description 根據產品 ID 軟刪除產品，需要 JWT 認證
// @Tags 產品
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "產品 ID" example(1)
// @Success 200 {object} map[string]string "刪除成功"
// @Failure 400 {object} map[string]string "無效的產品 ID"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 404 {object} map[string]string "產品不存在"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /product/{id} [delete]
func DeleteProduct(c *gin.Context) {
	if productDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection not configured"})
		return
	}

	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	// 獲取當前用戶 ID
	userID, exists := c.Get("user_id")
	var deleterID uint
	if exists {
		if id, ok := userID.(uint); ok {
			deleterID = id
		}
	}

	// 使用 Service 層
	svc := services.NewProductService(productDB)
	if err := svc.DeleteProduct(uint(productID), deleterID); err != nil {
		if err.Error() == "產品不存在或已被刪除" {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "product deleted successfully"})
}
