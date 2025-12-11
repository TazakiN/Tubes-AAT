package service

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"report-service/config"
	"report-service/internal/model"
	"report-service/internal/repository"

	"github.com/google/uuid"
)

type ReportService struct {
	reportRepo *repository.ReportRepository
	anonConfig config.AnonymousConfig
}

func NewReportService(reportRepo *repository.ReportRepository, anonConfig config.AnonymousConfig) *ReportService {
	return &ReportService{
		reportRepo: reportRepo,
		anonConfig: anonConfig,
	}
}

// CreateReport creates a new report with anonymous hashing if needed
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

// GetReports returns reports based on user role and department
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

// GetMyReports returns reports created by the user (non-anonymous only)
func (s *ReportService) GetMyReports(userID string) (*model.ReportListResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	reports, err := s.reportRepo.FindByReporterID(uid)
	if err != nil {
		return nil, err
	}

	return &model.ReportListResponse{
		Reports: reports,
		Total:   len(reports),
	}, nil
}

// GetReportByID returns a single report, masking anonymous info
func (s *ReportService) GetReportByID(id uuid.UUID, userRole string, userID string, department *string) (*model.Report, error) {
	report, err := s.reportRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// RBAC check: Admin can only view reports in their department
	if userRole != "warga" && department != nil {
		if report.Category.Department != *department {
			return nil, err
		}
	}

	// Warga can only view their own reports or public reports
	if userRole == "warga" {
		if report.PrivacyLevel != model.PrivacyPublic {
			// Check if this is the reporter's own report
			if report.ReporterID == nil || report.ReporterID.String() != userID {
				return nil, err
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

// UpdateReportStatus updates the status of a report (admin only)
func (s *ReportService) UpdateReportStatus(reportID uuid.UUID, status model.ReportStatus, department *string) error {
	// Verify admin can access this report's category
	report, err := s.reportRepo.FindByID(reportID)
	if err != nil {
		return err
	}

	if department != nil && report.Category.Department != *department {
		return err
	}

	return s.reportRepo.UpdateStatus(reportID, status)
}

func (s *ReportService) hashUserID(userID string) string {
	data := userID + s.anonConfig.Salt
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
