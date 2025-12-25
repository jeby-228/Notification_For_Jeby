package services

import (
	"errors"
	"member_API/models"
	"time"

	"gorm.io/gorm"
)

type ProductService struct {
	DB *gorm.DB
}

func NewProductService(db *gorm.DB) *ProductService {
	return &ProductService{DB: db}
}

// CreateProduct 建立新產品
func (s *ProductService) CreateProduct(name string, price float64, description, image string, stock int, creatorId uint) (*models.Product, error) {
	now := time.Now()
	product := &models.Product{
		Base: models.Base{
			CreationTime: now,
			CreatorId:    creatorId,
			IsDeleted:    false,
		},
		ProductName:        name,
		ProductPrice:       price,
		ProductDescription: description,
		ProductImage:       image,
		ProductStock:       stock,
	}

	if err := s.DB.Create(product).Error; err != nil {
		return nil, err
	}

	return product, nil
}

// UpdateProduct 更新產品資訊
func (s *ProductService) UpdateProduct(id uint, updates map[string]interface{}, modifierId uint) (*models.Product, error) {
	var product models.Product
	if err := s.DB.Where("is_deleted = ?", false).First(&product, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("產品不存在")
		}
		return nil, err
	}

	now := time.Now()
	updates["last_modification_time"] = &now
	updates["last_modifier_id"] = modifierId

	if err := s.DB.Model(&product).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新載入產品資料
	if err := s.DB.First(&product, id).Error; err != nil {
		return nil, err
	}

	return &product, nil
}

// DeleteProduct 軟刪除產品
func (s *ProductService) DeleteProduct(id uint, deleterId uint) error {
	now := time.Now()
	result := s.DB.Model(&models.Product{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(map[string]interface{}{
			"is_deleted":             true,
			"deleted_at":             &now,
			"last_modifier_id":       deleterId,
			"last_modification_time": &now,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("產品不存在或已被刪除")
	}

	return nil
}

// GetProductByID 取得單一產品
func (s *ProductService) GetProductByID(id uint) (*models.Product, error) {
	var product models.Product
	if err := s.DB.Where("is_deleted = ?", false).First(&product, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("產品不存在")
		}
		return nil, err
	}
	return &product, nil
}

// GetProducts 取得產品列表（支持分頁）
func (s *ProductService) GetProducts(limit, offset int) ([]models.Product, int64, error) {
	var products []models.Product
	var total int64

	// 取得總數
	if err := s.DB.Model(&models.Product{}).Where("is_deleted = ?", false).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 取得產品列表
	if err := s.DB.Where("is_deleted = ?", false).
		Order("sort ASC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}
