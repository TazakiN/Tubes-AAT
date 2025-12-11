package handler

import (
	"net/http"
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

// CreateReport handles POST /
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

	// Validate privacy level
	validLevels := map[model.PrivacyLevel]bool{
		model.PrivacyPublic:    true,
		model.PrivacyPrivate:   true,
		model.PrivacyAnonymous: true,
	}
	if !validLevels[req.PrivacyLevel] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid privacy level"})
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

// GetReports handles GET /
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

// GetMyReports handles GET /my - returns user's own non-anonymous reports
func (h *ReportHandler) GetMyReports(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	response, err := h.reportService.GetMyReports(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetReportByID handles GET /:id
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

// UpdateStatus handles PATCH /:id/status
func (h *ReportHandler) UpdateStatus(c *gin.Context) {
	userRole := c.GetHeader("X-User-Role")
	userDept := c.GetHeader("X-User-Department")

	// Only admin can update status
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

	// Validate status
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

// Health check endpoint
func (h *ReportHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}
