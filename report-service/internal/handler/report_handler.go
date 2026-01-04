package handler

import (
	"net/http"
	"strconv"
	"strings"

	"report-service/internal/model"
	"report-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReportHandler struct {
	reportService *service.ReportService
}

func NewReportHandler(reportService *service.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

func (h *ReportHandler) CreateReport(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	userName := c.GetHeader("X-User-Name")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req model.CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validLevels := map[model.PrivacyLevel]bool{
		model.PrivacyPublic:    true,
		model.PrivacyPrivate:   true,
		model.PrivacyAnonymous: true,
	}
	if !validLevels[req.PrivacyLevel] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid privacy level"})
		return
	}

	if req.NewCategoryName != nil && *req.NewCategoryName != "" {
		if req.NewCategoryDepartment == nil || *req.NewCategoryDepartment == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "new_category_department is required when creating new category"})
			return
		}
		category, err := h.reportService.GetOrCreateCategory(*req.NewCategoryName, *req.NewCategoryDepartment)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create category: " + err.Error()})
			return
		}
		req.CategoryID = category.ID
	}

	if req.CategoryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id or new_category_name is required"})
		return
	}

	report, err := h.reportService.CreateReport(&req, userID, userName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Report created successfully",
		"report":  report,
	})
}

func (h *ReportHandler) GetPublicReports(c *gin.Context) {
	search := c.Query("search")
	categoryIDStr := c.Query("category_id")

	var categoryID *int
	if categoryIDStr != "" {
		id, err := strconv.Atoi(categoryIDStr)
		if err == nil && id > 0 {
			categoryID = &id
		}
	}

	if search != "" || categoryID != nil {
		response, err := h.reportService.SearchPublicReports(search, categoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, response)
		return
	}

	response, err := h.reportService.GetPublicReports()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *ReportHandler) GetReports(c *gin.Context) {
	userRole := c.GetHeader("X-User-Role")
	userDept := c.GetHeader("X-User-Department")

	if userRole == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var department *string
	if userDept != "" {
		department = &userDept
	}

	response, err := h.reportService.GetReports(userRole, department)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *ReportHandler) GetMyReports(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	search := c.Query("search")
	categoryIDStr := c.Query("category_id")

	var categoryID *int
	if categoryIDStr != "" {
		id, err := strconv.Atoi(categoryIDStr)
		if err == nil && id > 0 {
			categoryID = &id
		}
	}

	if search != "" || categoryID != nil {
		response, err := h.reportService.SearchMyReports(userID, search, categoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, response)
		return
	}

	response, err := h.reportService.GetMyReports(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *ReportHandler) GetReportByID(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	userRole := c.GetHeader("X-User-Role")
	userDept := c.GetHeader("X-User-Department")

	if userRole == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	reportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}

	var department *string
	if userDept != "" {
		department = &userDept
	}

	report, err := h.reportService.GetReportByID(reportID, userRole, userID, department)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found or access denied"})
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *ReportHandler) UpdateStatus(c *gin.Context) {
	userRole := c.GetHeader("X-User-Role")
	userDept := c.GetHeader("X-User-Department")

	if !strings.HasPrefix(userRole, "admin_") {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can update report status"})
		return
	}

	reportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}

	var req model.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validStatuses := map[model.ReportStatus]bool{
		model.StatusPending:    true,
		model.StatusAccepted:   true,
		model.StatusInProgress: true,
		model.StatusCompleted:  true,
		model.StatusRejected:   true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	var department *string
	if userDept != "" {
		department = &userDept
	}

	if err := h.reportService.UpdateReportStatus(reportID, req.Status, department); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found or access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

func (h *ReportHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func (h *ReportHandler) UpdateReport(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	reportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}

	var req model.UpdateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Title == nil && req.Description == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nothing to update"})
		return
	}

	report, err := h.reportService.UpdateReport(reportID, userID, &req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Report updated successfully",
		"report":  report,
	})
}

func (h *ReportHandler) GetCategories(c *gin.Context) {
	categories, err := h.reportService.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

func (h *ReportHandler) CreateCategory(c *gin.Context) {
	var req model.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validDepts := map[string]bool{
		"kebersihan":    true,
		"kesehatan":     true,
		"infrastruktur": true,
	}
	if !validDepts[strings.ToLower(req.Department)] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department, must be kebersihan, kesehatan, or infrastruktur"})
		return
	}

	category, err := h.reportService.GetOrCreateCategory(req.Name, strings.ToLower(req.Department))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Category created successfully",
		"category": category,
	})
}
