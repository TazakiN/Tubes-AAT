package repository

import (
	"database/sql"
	"fmt"

	"report-service/internal/model"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(report *model.Report) error {
	query := `
		INSERT INTO reports (id, title, description, category_id, location_lat, location_lng, 
			photo_url, privacy_level, reporter_id, reporter_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Exec(query,
		report.ID,
		report.Title,
		report.Description,
		report.CategoryID,
		report.LocationLat,
		report.LocationLng,
		report.PhotoURL,
		report.PrivacyLevel,
		report.ReporterID,
		report.ReporterHash,
		report.Status,
		report.CreatedAt,
		report.UpdatedAt,
	)
	return err
}

func (r *ReportRepository) FindByID(id uuid.UUID) (*model.Report, error) {
	query := `
		SELECT r.id, r.title, r.description, r.category_id, r.location_lat, r.location_lng,
			r.photo_url, r.privacy_level, r.reporter_id, r.status, r.created_at, r.updated_at,
			c.id, c.name, c.department
		FROM reports r
		JOIN categories c ON r.category_id = c.id
		WHERE r.id = $1
	`
	report := &model.Report{Category: &model.Category{}}
	var lat, lng sql.NullFloat64
	var photoURL sql.NullString
	var reporterID sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&report.ID,
		&report.Title,
		&report.Description,
		&report.CategoryID,
		&lat,
		&lng,
		&photoURL,
		&report.PrivacyLevel,
		&reporterID,
		&report.Status,
		&report.CreatedAt,
		&report.UpdatedAt,
		&report.Category.ID,
		&report.Category.Name,
		&report.Category.Department,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("report not found")
		}
		return nil, err
	}

	if lat.Valid {
		report.LocationLat = &lat.Float64
	}
	if lng.Valid {
		report.LocationLng = &lng.Float64
	}
	if photoURL.Valid {
		report.PhotoURL = &photoURL.String
	}
	if reporterID.Valid {
		uid, _ := uuid.Parse(reporterID.String)
		report.ReporterID = &uid
	}

	return report, nil
}

// FindAll returns reports filtered by department (for admin) or all public reports (for warga)
func (r *ReportRepository) FindAll(userRole string, department *string) ([]model.Report, error) {
	var query string
	var args []interface{}

	// Warga can see: their own reports + public reports
	// Admin Dinas X can see: reports in category belonging to their department
	if userRole == "warga" {
		query = `
			SELECT r.id, r.title, r.description, r.category_id, r.location_lat, r.location_lng,
				r.photo_url, r.privacy_level, r.reporter_id, r.status, r.created_at, r.updated_at,
				c.id, c.name, c.department
			FROM reports r
			JOIN categories c ON r.category_id = c.id
			WHERE r.privacy_level = 'public'
			ORDER BY r.created_at DESC
		`
	} else if department != nil {
		// Admin hanya bisa lihat laporan di departemennya
		query = `
			SELECT r.id, r.title, r.description, r.category_id, r.location_lat, r.location_lng,
				r.photo_url, r.privacy_level, r.reporter_id, r.status, r.created_at, r.updated_at,
				c.id, c.name, c.department
			FROM reports r
			JOIN categories c ON r.category_id = c.id
			WHERE c.department = $1
			ORDER BY r.created_at DESC
		`
		args = append(args, *department)
	} else {
		return nil, fmt.Errorf("unauthorized access")
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.Report
	for rows.Next() {
		report := model.Report{Category: &model.Category{}}
		var lat, lng sql.NullFloat64
		var photoURL sql.NullString
		var reporterID sql.NullString

		err := rows.Scan(
			&report.ID,
			&report.Title,
			&report.Description,
			&report.CategoryID,
			&lat,
			&lng,
			&photoURL,
			&report.PrivacyLevel,
			&reporterID,
			&report.Status,
			&report.CreatedAt,
			&report.UpdatedAt,
			&report.Category.ID,
			&report.Category.Name,
			&report.Category.Department,
		)
		if err != nil {
			return nil, err
		}

		if lat.Valid {
			report.LocationLat = &lat.Float64
		}
		if lng.Valid {
			report.LocationLng = &lng.Float64
		}
		if photoURL.Valid {
			report.PhotoURL = &photoURL.String
		}
		if reporterID.Valid {
			uid, _ := uuid.Parse(reporterID.String)
			report.ReporterID = &uid
		}

		reports = append(reports, report)
	}

	return reports, nil
}

// FindByReporterID returns reports created by a specific user
func (r *ReportRepository) FindByReporterID(reporterID uuid.UUID) ([]model.Report, error) {
	query := `
		SELECT r.id, r.title, r.description, r.category_id, r.location_lat, r.location_lng,
			r.photo_url, r.privacy_level, r.reporter_id, r.status, r.created_at, r.updated_at,
			c.id, c.name, c.department
		FROM reports r
		JOIN categories c ON r.category_id = c.id
		WHERE r.reporter_id = $1
		ORDER BY r.created_at DESC
	`

	rows, err := r.db.Query(query, reporterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.Report
	for rows.Next() {
		report := model.Report{Category: &model.Category{}}
		var lat, lng sql.NullFloat64
		var photoURL sql.NullString
		var reporterIDNull sql.NullString

		err := rows.Scan(
			&report.ID,
			&report.Title,
			&report.Description,
			&report.CategoryID,
			&lat,
			&lng,
			&photoURL,
			&report.PrivacyLevel,
			&reporterIDNull,
			&report.Status,
			&report.CreatedAt,
			&report.UpdatedAt,
			&report.Category.ID,
			&report.Category.Name,
			&report.Category.Department,
		)
		if err != nil {
			return nil, err
		}

		if lat.Valid {
			report.LocationLat = &lat.Float64
		}
		if lng.Valid {
			report.LocationLng = &lng.Float64
		}
		if photoURL.Valid {
			report.PhotoURL = &photoURL.String
		}
		if reporterIDNull.Valid {
			uid, _ := uuid.Parse(reporterIDNull.String)
			report.ReporterID = &uid
		}

		reports = append(reports, report)
	}

	return reports, nil
}

func (r *ReportRepository) UpdateStatus(id uuid.UUID, status model.ReportStatus) error {
	query := `UPDATE reports SET status = $1, updated_at = NOW() WHERE id = $2`
	result, err := r.db.Exec(query, status, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("report not found")
	}

	return nil
}

func (r *ReportRepository) GetCategoryByID(id int) (*model.Category, error) {
	query := `SELECT id, name, department FROM categories WHERE id = $1`
	cat := &model.Category{}
	err := r.db.QueryRow(query, id).Scan(&cat.ID, &cat.Name, &cat.Department)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("category not found")
		}
		return nil, err
	}
	return cat, nil
}
