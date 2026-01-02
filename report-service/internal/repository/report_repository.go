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

func (r *ReportRepository) GetDB() *sql.DB {
	return r.db
}

func (r *ReportRepository) FindByID(id uuid.UUID) (*model.Report, error) {
	query := `
		SELECT r.id, r.title, r.description, r.category_id, r.location_lat, r.location_lng,
			r.photo_url, r.privacy_level, r.reporter_id, r.status, r.vote_score, r.created_at, r.updated_at,
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
		&report.VoteScore,
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

// Returns reports filtered by department (for admin) or all public reports (for warga).
func (r *ReportRepository) FindAll(userRole string, department *string) ([]model.Report, error) {
	var query string
	var args []interface{}

	// Warga can see: their own reports + public reports
	// Admin Dinas X can see: reports in category belonging to their department
	if userRole == "warga" {
		query = `
			SELECT r.id, r.title, r.description, r.category_id, r.location_lat, r.location_lng,
				r.photo_url, r.privacy_level, r.reporter_id, r.status, r.vote_score, r.created_at, r.updated_at,
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
				r.photo_url, r.privacy_level, r.reporter_id, r.status, r.vote_score, r.created_at, r.updated_at,
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
			&report.VoteScore,
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

// Returns all reports created by a specific user.
func (r *ReportRepository) FindByReporterID(reporterID uuid.UUID) ([]model.Report, error) {
	query := `
		SELECT r.id, r.title, r.description, r.category_id, r.location_lat, r.location_lng,
			r.photo_url, r.privacy_level, r.reporter_id, r.status, r.vote_score, r.created_at, r.updated_at,
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
			&report.VoteScore,
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

// Returns all public reports with vote scores, ordered by score then creation time.
func (r *ReportRepository) GetPublicReports() ([]model.Report, error) {
	query := `
		SELECT r.id, r.title, r.description, r.category_id, r.location_lat, r.location_lng,
			r.photo_url, r.privacy_level, r.reporter_id, r.status, r.vote_score, r.created_at, r.updated_at,
			c.id, c.name, c.department,
			u.name as reporter_name
		FROM reports r
		JOIN categories c ON r.category_id = c.id
		LEFT JOIN users u ON r.reporter_id = u.id
		WHERE r.privacy_level = 'public'
		ORDER BY r.vote_score DESC, r.created_at DESC
	`

	rows, err := r.db.Query(query)
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
		var reporterName sql.NullString

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
			&report.VoteScore,
			&report.CreatedAt,
			&report.UpdatedAt,
			&report.Category.ID,
			&report.Category.Name,
			&report.Category.Department,
			&reporterName,
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
		if reporterName.Valid {
			report.ReporterName = &reporterName.String
		}

		reports = append(reports, report)
	}

	return reports, nil
}

// Dynamically updates title and/or description fields on a report.
func (r *ReportRepository) Update(id uuid.UUID, title, description *string) error {
	// Build dynamic query based on what fields are provided
	query := `UPDATE reports SET updated_at = NOW()`
	args := []interface{}{}
	argIndex := 1

	if title != nil {
		query += fmt.Sprintf(", title = $%d", argIndex)
		args = append(args, *title)
		argIndex++
	}
	if description != nil {
		query += fmt.Sprintf(", description = $%d", argIndex)
		args = append(args, *description)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, id)

	result, err := r.db.Exec(query, args...)
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

// Returns all categories ordered by department and name.
func (r *ReportRepository) GetAllCategories() ([]model.Category, error) {
	query := `SELECT id, name, department FROM categories ORDER BY department, name`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []model.Category
	for rows.Next() {
		var cat model.Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Department); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

// Performs full-text search on public reports with optional category filtering.
func (r *ReportRepository) SearchPublicReports(search string, categoryID *int) ([]model.Report, error) {
	query := `
		SELECT r.id, r.title, r.description, r.category_id, r.location_lat, r.location_lng,
			r.photo_url, r.privacy_level, r.reporter_id, r.status, r.vote_score, r.created_at, r.updated_at,
			c.id, c.name, c.department,
			u.name as reporter_name
		FROM reports r
		JOIN categories c ON r.category_id = c.id
		LEFT JOIN users u ON r.reporter_id = u.id
		WHERE r.privacy_level = 'public'
	`
	args := []interface{}{}
	argIndex := 1

	if search != "" {
		query += fmt.Sprintf(" AND (LOWER(r.title) LIKE LOWER($%d) OR LOWER(r.description) LIKE LOWER($%d))", argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if categoryID != nil && *categoryID > 0 {
		query += fmt.Sprintf(" AND r.category_id = $%d", argIndex)
		args = append(args, *categoryID)
		argIndex++
	}

	query += " ORDER BY r.vote_score DESC, r.created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanReportsWithReporterName(rows)
}

// Performs full-text search on user's own reports with optional category filtering.
func (r *ReportRepository) SearchMyReports(reporterID uuid.UUID, search string, categoryID *int) ([]model.Report, error) {
	query := `
		SELECT r.id, r.title, r.description, r.category_id, r.location_lat, r.location_lng,
			r.photo_url, r.privacy_level, r.reporter_id, r.status, r.vote_score, r.created_at, r.updated_at,
			c.id, c.name, c.department
		FROM reports r
		JOIN categories c ON r.category_id = c.id
		WHERE r.reporter_id = $1
	`
	args := []interface{}{reporterID}
	argIndex := 2

	if search != "" {
		query += fmt.Sprintf(" AND (LOWER(r.title) LIKE LOWER($%d) OR LOWER(r.description) LIKE LOWER($%d))", argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if categoryID != nil && *categoryID > 0 {
		query += fmt.Sprintf(" AND r.category_id = $%d", argIndex)
		args = append(args, *categoryID)
		argIndex++
	}

	query += " ORDER BY r.created_at DESC"

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
			&report.VoteScore,
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

// Helper to scan report rows that include reporter name from users table.
func (r *ReportRepository) scanReportsWithReporterName(rows *sql.Rows) ([]model.Report, error) {
	var reports []model.Report
	for rows.Next() {
		report := model.Report{Category: &model.Category{}}
		var lat, lng sql.NullFloat64
		var photoURL sql.NullString
		var reporterID sql.NullString
		var reporterName sql.NullString

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
			&report.VoteScore,
			&report.CreatedAt,
			&report.UpdatedAt,
			&report.Category.ID,
			&report.Category.Name,
			&report.Category.Department,
			&reporterName,
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
		if reporterName.Valid {
			report.ReporterName = &reporterName.String
		}

		reports = append(reports, report)
	}

	return reports, nil
}

// Performs case-insensitive lookup of category by name.
func (r *ReportRepository) FindCategoryByName(name string) (*model.Category, error) {
	query := `SELECT id, name, department FROM categories WHERE LOWER(name) = LOWER($1)`
	cat := &model.Category{}
	err := r.db.QueryRow(query, name).Scan(&cat.ID, &cat.Name, &cat.Department)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}
	return cat, nil
}

// Inserts a new category and returns it with the generated ID.
func (r *ReportRepository) CreateCategory(name, department string) (*model.Category, error) {
	query := `INSERT INTO categories (name, department) VALUES ($1, $2) RETURNING id`
	var id int
	err := r.db.QueryRow(query, name, department).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &model.Category{ID: id, Name: name, Department: department}, nil
}
