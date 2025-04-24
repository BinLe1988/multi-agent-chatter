package handlers

import (
	"multi-agent-chatter/database"
	"multi-agent-chatter/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetRechargePackages 获取充值套餐
func GetRechargePackages(c *gin.Context) {
	packages := models.GetDefaultRechargePackages()
	c.JSON(http.StatusOK, gin.H{
		"packages": packages,
	})
}

// CreateRechargeOrder 创建充值订单
func CreateRechargeOrder(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req models.RechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 计算充值积分和金额
	var credits int
	var amount float64

	if req.PackageID != "" {
		// 套餐充值
		packages := models.GetDefaultRechargePackages()
		var selectedPackage *models.RechargePackage

		for _, pkg := range packages {
			if pkg.ID == req.PackageID {
				selectedPackage = &pkg
				break
			}
		}

		if selectedPackage == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
			return
		}

		credits = selectedPackage.Credits
		amount = selectedPackage.Price
	} else if req.CustomAmount > 0 {
		// 自定义充值金额
		credits = req.CustomAmount
		// 计算价格：10元 = 100积分
		amount = float64(credits) / 100 * 10
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either package ID or custom amount is required"})
		return
	}

	// 创建订单
	orderNo := "R" + time.Now().Format("20060102") + uuid.New().String()[:8]
	payment := models.Payment{
		UserID:  userID.(uint),
		OrderNo: orderNo,
		Amount:  amount,
		Credits: credits,
		Method:  req.Method,
		Status:  models.PaymentPending,
	}

	if err := database.DB.Create(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment order"})
		return
	}

	// 这里应该调用实际的支付网关API获取支付链接或二维码
	// 这里仅作为演示，返回模拟数据
	paymentURL := "https://example.com/pay/" + orderNo
	paymentQR := "data:image/png;base64,..."

	c.JSON(http.StatusOK, gin.H{
		"payment": models.PaymentResponse{
			OrderNo:    payment.OrderNo,
			Amount:     payment.Amount,
			Credits:    payment.Credits,
			Method:     payment.Method,
			Status:     payment.Status,
			PaymentURL: paymentURL,
			PaymentQR:  paymentQR,
			CreatedAt:  payment.CreatedAt,
		},
	})
}

// CheckPaymentStatus 检查支付状态
func CheckPaymentStatus(c *gin.Context) {
	orderNo := c.Param("orderNo")

	var payment models.Payment
	if err := database.DB.Where("order_no = ?", orderNo).First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": payment.Status,
	})
}

// HandlePaymentCallback 处理支付回调
func HandlePaymentCallback(c *gin.Context) {
	// 支付网关的回调通常会包含订单号和支付状态等信息
	orderNo := c.PostForm("order_no")
	status := c.PostForm("status")
	transactionID := c.PostForm("transaction_id")

	var payment models.Payment
	if err := database.DB.Where("order_no = ?", orderNo).First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment order not found"})
		return
	}

	// 验证回调的合法性（实际应用中应该验证签名等）

	if status == "success" {
		now := time.Now()
		payment.Status = models.PaymentCompleted
		payment.CompletedAt = &now
		payment.TransactionID = transactionID

		// 更新用户积分
		var user models.User
		if err := database.DB.First(&user, payment.UserID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
			return
		}

		user.Credits += payment.Credits

		// 使用事务确保数据一致性
		tx := database.DB.Begin()
		if err := tx.Save(&payment).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment"})
			return
		}

		if err := tx.Save(&user).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user credits"})
			return
		}

		tx.Commit()
	} else {
		payment.Status = models.PaymentFailed
		database.DB.Save(&payment)
	}

	// 返回给支付网关的响应
	c.String(http.StatusOK, "success")
}

// GetPaymentHistory 获取支付历史
func GetPaymentHistory(c *gin.Context) {
	userID, _ := c.Get("userID")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	var payments []models.Payment
	var count int64

	database.DB.Model(&models.Payment{}).Where("user_id = ?", userID).Count(&count)
	database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&payments)

	c.JSON(http.StatusOK, gin.H{
		"total":    count,
		"page":     page,
		"pageSize": pageSize,
		"payments": payments,
	})
}
