package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"report-service/config"
	"report-service/internal/model"
	"report-service/internal/repository"

	"github.com/google/uuid"
)

type ReportService struct {
	reportRepo          *repository.ReportRepository
	anonConfig          config.AnonymousConfig
	notificationService *NotificationService
}

func NewReportService(reportRepo *repository.ReportRepository, anonConfig config.AnonymousConfig) *ReportService {
	return &ReportService{
		reportRepo: reportRepo,
		anonConfig: anonConfig,
	}
}

// Injects the notification service dependency for status update notifications.
func (s *ReportService) SetNotificationService(ns *NotificationService) {
	s.notificationService = ns
}

// Persists a new report. Applies SHA-256 hashing to reporter ID for anonymous reports.
func (s *ReportService) CreateReport(req *model.CreateReportRequest, userID string, userName string) (*model.Report, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	report := &model.Report{
		ID:           uuid.New(),
		Title:        req.Title,
		Description:  req.Description,
		CategoryID:   req.CategoryID,
		LocationLat:  req.LocationLat,
		LocationLng:  req.LocationLng,
		PhotoURL:     req.PhotoURL,
		PrivacyLevel: req.PrivacyLevel,
		Status:       model.StatusPending,
		VoteScore:    0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Handle privacy level
	switch req.PrivacyLevel {
	case model.PrivacyAnonymous:
		// Hash the user ID with salt - cannot be reversed
		hash := s.hashUserID(userID)
		report.ReporterHash = &hash
		// ReporterID stays nil for anonymous
	default:
		report.ReporterID = &uid
		report.ReporterName = &userName
	}

	if err := s.reportRepo.Create(report); err != nil {
		return nil, err
	}

	// Clear hash from response
	report.ReporterHash = nil

	return report, nil
}

// Returns reports filtered by user role and department. Masks anonymous reporter info.
func (s *ReportService) GetReports(userRole string, department *string) (*model.ReportListResponse, error) {
	reports, err := s.reportRepo.FindAll(userRole, department)
	if err != nil {
		return nil, err
	}

	// Mask reporter info for anonymous reports
	for i := range reports {
		if reports[i].PrivacyLevel == model.PrivacyAnonymous {
			reports[i].ReporterID = nil
			reports[i].ReporterName = nil
		}
	}

	return &model.ReportListResponse{
		Reports: reports,
		Total:   len(reports),
	}, nil
}

// Returns all public reports. No authentication required.
func (s *ReportService) GetPublicReports() (*model.ReportListResponse, error) {
	reports, err := s.reportRepo.GetPublicReports()
	if err != nil {
		return nil, err
	}

	if reports == nil {
		reports = []model.Report{}
	}

	return &model.ReportListResponse{
		Reports: reports,
		Total:   len(reports),
	}, nil
}

// Returns non-anonymous reports created by the specified user.
func (s *ReportService) GetMyReports(userID string) (*model.ReportListResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	reports, err := s.reportRepo.FindByReporterID(uid)
	if err != nil {
		return nil, err
	}

	if reports == nil {
		reports = []model.Report{}
	}

	return &model.ReportListResponse{
		Reports: reports,
		Total:   len(reports),
	}, nil
}

// Retrieves a single report with RBAC validation. Masks anonymous reporter info.
func (s *ReportService) GetReportByID(id uuid.UUID, userRole string, userID string, department *string) (*model.Report, error) {
	report, err := s.reportRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// RBAC check: Admin can only view reports in their department
	if userRole != "warga" && department != nil {
		if report.Category.Department != *department {
			return nil, fmt.Errorf("access denied")
		}
	}

	// Warga can only view their own reports or public reports
	if userRole == "warga" {
		if report.PrivacyLevel != model.PrivacyPublic {
			// Check if this is the reporter's own report
			if report.ReporterID == nil || report.ReporterID.String() != userID {
				return nil, fmt.Errorf("access denied")
			}
		}
	}

	// Mask anonymous reporter info
	if report.PrivacyLevel == model.PrivacyAnonymous {
		report.ReporterID = nil
		report.ReporterName = nil
	}

	return report, nil
}

// Updates title and/or description. Only the report owner can perform this action.
func (s *ReportService) UpdateReport(reportID uuid.UUID, userID string, req *model.UpdateReportRequest) (*model.Report, error) {
	// Get the report first to check ownership
	report, err := s.reportRepo.FindByID(reportID)
	if err != nil {
		return nil, err
	}

	// Check ownership - only the reporter can edit
	if report.ReporterID == nil || report.ReporterID.String() != userID {
		return nil, fmt.Errorf("only the report owner can edit")
	}

	// Update the report
	if err := s.reportRepo.Update(reportID, req.Title, req.Description); err != nil {
		return nil, err
	}

	// Return updated report
	return s.reportRepo.FindByID(reportID)
}

// Updates status with department validation. Triggers async notification to reporter.
func (s *ReportService) UpdateReportStatus(reportID uuid.UUID, status model.ReportStatus, department *string) error {
	// Verify admin can access this report's category
	report, err := s.reportRepo.FindByID(reportID)
	if err != nil {
		return err
	}

	if department != nil && report.Category.Department != *department {
		return fmt.Errorf("access denied")
	}

	// Update status
	if err := s.reportRepo.UpdateStatus(reportID, status); err != nil {
		return err
	}

	// Send notification to the reporter if notification service is available
	if s.notificationService != nil {
		go func() {
			err := s.notificationService.CreateStatusNotification(reportID, status, report.Title, report.ReporterID)
			if err != nil {
				// Log error but don't fail the status update
				fmt.Printf("Failed to create notification: %v\n", err)
			}
		}()
	}

	return nil
}

// Returns all available report categories.
func (s *ReportService) GetCategories() ([]model.Category, error) {
	return s.reportRepo.GetAllCategories()
}

// Searches public reports with optional text search and category filtering.
func (s *ReportService) SearchPublicReports(search string, categoryID *int) (*model.ReportListResponse, error) {
	reports, err := s.reportRepo.SearchPublicReports(search, categoryID)
	if err != nil {
		return nil, err
	}

	if reports == nil {
		reports = []model.Report{}
	}

	return &model.ReportListResponse{
		Reports: reports,
		Total:   len(reports),
	}, nil
}

// Searches user's own reports with optional text search and category filtering.
func (s *ReportService) SearchMyReports(userID string, search string, categoryID *int) (*model.ReportListResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	reports, err := s.reportRepo.SearchMyReports(uid, search, categoryID)
	if err != nil {
		return nil, err
	}

	if reports == nil {
		reports = []model.Report{}
	}

	return &model.ReportListResponse{
		Reports: reports,
		Total:   len(reports),
	}, nil
}

// Returns existing category by case-insensitive name match, or creates a new one.
func (s *ReportService) GetOrCreateCategory(name, department string) (*model.Category, error) {
	// Try to find existing category
	existing, err := s.reportRepo.FindCategoryByName(name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// Create new category
	return s.reportRepo.CreateCategory(name, department)
}

func (s *ReportService) hashUserID(userID string) string {
	data := userID + s.anonConfig.Salt
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
