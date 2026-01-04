package service

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"report-service/config"
	"report-service/internal/messaging"
	"report-service/internal/model"
	"report-service/internal/repository"

	"github.com/google/uuid"
)

type ReportService struct {
	reportRepo *repository.ReportRepository
	outboxRepo *repository.OutboxRepository
	anonConfig config.AnonymousConfig
	rmq        *messaging.RabbitMQ
	db         *sql.DB
}

func NewReportService(reportRepo *repository.ReportRepository, outboxRepo *repository.OutboxRepository, anonConfig config.AnonymousConfig, rmq *messaging.RabbitMQ, db *sql.DB) *ReportService {
	return &ReportService{
		reportRepo: reportRepo,
		outboxRepo: outboxRepo,
		anonConfig: anonConfig,
		rmq:        rmq,
		db:         db,
	}
}

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

	switch req.PrivacyLevel {
	case model.PrivacyAnonymous:
		hash := s.hashUserID(userID)
		report.ReporterHash = &hash
	default:
		report.ReporterID = &uid
		report.ReporterName = &userName
	}

	if err := s.reportRepo.Create(report); err != nil {
		return nil, err
	}

	if s.outboxRepo != nil {
		reporterIDStr := ""
		if report.ReporterID != nil {
			reporterIDStr = report.ReporterID.String()
		}
		reporterNameStr := ""
		if report.ReporterName != nil {
			reporterNameStr = *report.ReporterName
		}

		msg := messaging.ReportCreatedMessage{
			ReportID:     report.ID.String(),
			ReportTitle:  report.Title,
			CategoryID:   report.CategoryID,
			ReporterID:   reporterIDStr,
			ReporterName: reporterNameStr,
			PrivacyLevel: string(report.PrivacyLevel),
			Timestamp:    time.Now().Unix(),
		}

		if err := s.outboxRepo.Create(messaging.RoutingKeyReportCreated, msg); err != nil {
			log.Printf("outbox save failed: %v", err)
		}
	}

	report.ReporterHash = nil

	return report, nil
}

func (s *ReportService) GetReports(userRole string, department *string) (*model.ReportListResponse, error) {
	reports, err := s.reportRepo.FindAll(userRole, department)
	if err != nil {
		return nil, err
	}

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

func (s *ReportService) GetReportByID(id uuid.UUID, userRole string, userID string, department *string) (*model.Report, error) {
	report, err := s.reportRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if userRole != "warga" && department != nil {
		if report.Category.Department != *department {
			return nil, fmt.Errorf("access denied")
		}
	}

	if userRole == "warga" {
		if report.PrivacyLevel != model.PrivacyPublic {
			if report.ReporterID == nil || report.ReporterID.String() != userID {
				return nil, fmt.Errorf("access denied")
			}
		}
	}

	if report.PrivacyLevel == model.PrivacyAnonymous {
		report.ReporterID = nil
		report.ReporterName = nil
	}

	return report, nil
}

func (s *ReportService) UpdateReport(reportID uuid.UUID, userID string, req *model.UpdateReportRequest) (*model.Report, error) {
	report, err := s.reportRepo.FindByID(reportID)
	if err != nil {
		return nil, err
	}

	if report.ReporterID == nil || report.ReporterID.String() != userID {
		return nil, fmt.Errorf("unauthorized: bukan pemilik laporan")
	}

	if err := s.reportRepo.Update(reportID, req.Title, req.Description); err != nil {
		return nil, err
	}

	return s.reportRepo.FindByID(reportID)
}

func (s *ReportService) UpdateReportStatus(reportID uuid.UUID, status model.ReportStatus, department *string) error {
	report, err := s.reportRepo.FindByID(reportID)
	if err != nil {
		return err
	}

	if department != nil && report.Category.Department != *department {
		return fmt.Errorf("access denied")
	}

	if err := s.reportRepo.UpdateStatus(reportID, status); err != nil {
		return err
	}

	if s.outboxRepo != nil {
		msg := messaging.StatusUpdateMessage{
			ReportID:    reportID.String(),
			ReportTitle: report.Title,
			NewStatus:   string(status),
			Timestamp:   time.Now().Unix(),
		}

		if report.ReporterID != nil {
			msg.ReporterID = report.ReporterID.String()
		}

		if err := s.outboxRepo.Create(messaging.RoutingKeyStatusUpdate, msg); err != nil {
			log.Printf("outbox save failed: %v", err)
		}
	}

	return nil
}

func (s *ReportService) GetCategories() ([]model.Category, error) {
	return s.reportRepo.GetAllCategories()
}

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

func (s *ReportService) GetOrCreateCategory(name, department string) (*model.Category, error) {
	existing, err := s.reportRepo.FindCategoryByName(name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	return s.reportRepo.CreateCategory(name, department)
}

func (s *ReportService) hashUserID(userID string) string {
	data := userID + s.anonConfig.Salt
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
