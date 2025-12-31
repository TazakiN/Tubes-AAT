package repository

import (
	"database/sql"
	"fmt"
	"time"

	"report-service/internal/model"

	"github.com/google/uuid"
)

// Data access layer for report vote operations.
type VoteRepository struct {
	db *sql.DB
}

// Constructor for VoteRepository.
func NewVoteRepository(db *sql.DB) *VoteRepository {
	return &VoteRepository{db: db}
}

// Retrieves a user's existing vote on a report.
// Returns nil with no error if no vote exists.
func (r *VoteRepository) GetVote(reportID, userID uuid.UUID) (*model.ReportVote, error) {
	query := `
		SELECT id, report_id, user_id, vote_type, created_at
		FROM report_votes
		WHERE report_id = $1 AND user_id = $2
	`
	vote := &model.ReportVote{}
	err := r.db.QueryRow(query, reportID, userID).Scan(
		&vote.ID,
		&vote.ReportID,
		&vote.UserID,
		&vote.VoteType,
		&vote.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No vote exists, not an error
		}
		return nil, err
	}
	return vote, nil
}

// Inserts a new vote record.
func (r *VoteRepository) CreateVote(vote *model.ReportVote) error {
	query := `
		INSERT INTO report_votes (id, report_id, user_id, vote_type, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(query,
		vote.ID,
		vote.ReportID,
		vote.UserID,
		vote.VoteType,
		vote.CreatedAt,
	)
	return err
}

// Changes the vote type for an existing vote record.
func (r *VoteRepository) UpdateVote(voteID uuid.UUID, voteType model.VoteType) error {
	query := `UPDATE report_votes SET vote_type = $1 WHERE id = $2`
	result, err := r.db.Exec(query, voteType, voteID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("vote not found")
	}
	return nil
}

// Removes a vote record by report and user ID.
func (r *VoteRepository) DeleteVote(reportID, userID uuid.UUID) error {
	query := `DELETE FROM report_votes WHERE report_id = $1 AND user_id = $2`
	result, err := r.db.Exec(query, reportID, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("vote not found")
	}
	return nil
}

// Increments or decrements the vote_score column on a report.
// Delta can be positive (upvote) or negative (downvote).
func (r *VoteRepository) UpdateReportVoteScore(reportID uuid.UUID, delta int) error {
	query := `UPDATE reports SET vote_score = vote_score + $1 WHERE id = $2`
	result, err := r.db.Exec(query, delta, reportID)
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

// Returns the current vote_score value for a report.
func (r *VoteRepository) GetReportVoteScore(reportID uuid.UUID) (int, error) {
	query := `SELECT vote_score FROM reports WHERE id = $1`
	var score int
	err := r.db.QueryRow(query, reportID).Scan(&score)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("report not found")
		}
		return 0, err
	}
	return score, nil
}

// Atomically creates/updates vote and adjusts score within a transaction.
func (r *VoteRepository) VoteWithTransaction(reportID, userID uuid.UUID, voteType model.VoteType) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Check existing vote
	var existingVote *model.ReportVote
	query := `
		SELECT id, report_id, user_id, vote_type, created_at
		FROM report_votes
		WHERE report_id = $1 AND user_id = $2
		FOR UPDATE
	`
	existingVote = &model.ReportVote{}
	err = tx.QueryRow(query, reportID, userID).Scan(
		&existingVote.ID,
		&existingVote.ReportID,
		&existingVote.UserID,
		&existingVote.VoteType,
		&existingVote.CreatedAt,
	)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	hasExistingVote := err != sql.ErrNoRows

	var scoreDelta int

	if hasExistingVote {
		// User already voted
		if existingVote.VoteType == voteType {
			// Same vote type - no change needed
			var currentScore int
			err = tx.QueryRow(`SELECT vote_score FROM reports WHERE id = $1`, reportID).Scan(&currentScore)
			if err != nil {
				return 0, err
			}
			tx.Commit()
			return currentScore, nil
		}

		// Different vote type - update vote and calculate delta
		// If changing from upvote to downvote: delta = -2
		// If changing from downvote to upvote: delta = +2
		if voteType == model.VoteUpvote {
			scoreDelta = 2 // was downvote (-1), now upvote (+1), net change = +2
		} else {
			scoreDelta = -2 // was upvote (+1), now downvote (-1), net change = -2
		}

		_, err = tx.Exec(`UPDATE report_votes SET vote_type = $1 WHERE id = $2`, voteType, existingVote.ID)
		if err != nil {
			return 0, err
		}
	} else {
		// New vote
		if voteType == model.VoteUpvote {
			scoreDelta = 1
		} else {
			scoreDelta = -1
		}

		newVote := &model.ReportVote{
			ID:        uuid.New(),
			ReportID:  reportID,
			UserID:    userID,
			VoteType:  voteType,
			CreatedAt: time.Now(),
		}
		_, err = tx.Exec(`
			INSERT INTO report_votes (id, report_id, user_id, vote_type, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, newVote.ID, newVote.ReportID, newVote.UserID, newVote.VoteType, newVote.CreatedAt)
		if err != nil {
			return 0, err
		}
	}

	// Update report score
	_, err = tx.Exec(`UPDATE reports SET vote_score = vote_score + $1 WHERE id = $2`, scoreDelta, reportID)
	if err != nil {
		return 0, err
	}

	// Get new score
	var newScore int
	err = tx.QueryRow(`SELECT vote_score FROM reports WHERE id = $1`, reportID).Scan(&newScore)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return newScore, nil
}

// Atomically removes vote and reverses score adjustment within a transaction.
func (r *VoteRepository) RemoveVoteWithTransaction(reportID, userID uuid.UUID) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Get existing vote
	var voteType model.VoteType
	query := `SELECT vote_type FROM report_votes WHERE report_id = $1 AND user_id = $2 FOR UPDATE`
	err = tx.QueryRow(query, reportID, userID).Scan(&voteType)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no vote to remove")
		}
		return 0, err
	}

	// Calculate score delta (reverse the vote)
	var scoreDelta int
	if voteType == model.VoteUpvote {
		scoreDelta = -1
	} else {
		scoreDelta = 1
	}

	// Delete vote
	_, err = tx.Exec(`DELETE FROM report_votes WHERE report_id = $1 AND user_id = $2`, reportID, userID)
	if err != nil {
		return 0, err
	}

	// Update score
	_, err = tx.Exec(`UPDATE reports SET vote_score = vote_score + $1 WHERE id = $2`, scoreDelta, reportID)
	if err != nil {
		return 0, err
	}

	// Get new score
	var newScore int
	err = tx.QueryRow(`SELECT vote_score FROM reports WHERE id = $1`, reportID).Scan(&newScore)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return newScore, nil
}
