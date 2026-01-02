package service

import (
	"fmt"
	"log"
	"time"

	"report-service/internal/messaging"
	"report-service/internal/model"
	"report-service/internal/repository"

	"github.com/google/uuid"
)

// Business logic layer for voting operations on reports.
type VoteService struct {
	voteRepo   *repository.VoteRepository
	reportRepo *repository.ReportRepository
	rmq        *messaging.RabbitMQ
}

// Constructor for VoteService.
func NewVoteService(voteRepo *repository.VoteRepository, reportRepo *repository.ReportRepository, rmq *messaging.RabbitMQ) *VoteService {
	return &VoteService{
		voteRepo:   voteRepo,
		reportRepo: reportRepo,
		rmq:        rmq,
	}
}

// Registers or updates a vote. Returns new score and user's vote type.
// Only public reports can be voted on.
func (s *VoteService) CastVote(reportIDStr, userIDStr string, voteType model.VoteType) (*model.VoteResponse, error) {
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid report ID")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	// Validate vote type
	if voteType != model.VoteUpvote && voteType != model.VoteDownvote {
		return nil, fmt.Errorf("invalid vote type")
	}

	// Verify report exists and is public (only public reports can be voted on)
	report, err := s.reportRepo.FindByID(reportID)
	if err != nil {
		return nil, fmt.Errorf("report not found")
	}

	if report.PrivacyLevel != model.PrivacyPublic {
		return nil, fmt.Errorf("can only vote on public reports")
	}

	// Perform vote with transaction
	newScore, err := s.voteRepo.VoteWithTransaction(reportID, userID, voteType)
	if err != nil {
		return nil, err
	}

	// Publish to RabbitMQ asynchronously
	if s.rmq != nil {
		go func() {
			reporterIDStr := ""
			if report.ReporterID != nil {
				reporterIDStr = report.ReporterID.String()
			}

			msg := messaging.VoteReceivedMessage{
				ReportID:    reportID.String(),
				ReportTitle: report.Title,
				ReporterID:  reporterIDStr,
				VoterID:     userIDStr,
				VoteType:    string(voteType),
				NewScore:    newScore,
				Timestamp:   time.Now().Unix(),
			}
			if err := s.rmq.PublishVoteReceived(msg); err != nil {
				log.Printf("Failed to publish vote event: %v", err)
			}
		}()
	}

	return &model.VoteResponse{
		VoteScore:    newScore,
		UserVoteType: &voteType,
	}, nil
}

// Deletes user's vote and returns the updated score.
func (s *VoteService) RemoveVote(reportIDStr, userIDStr string) (*model.VoteResponse, error) {
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid report ID")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	newScore, err := s.voteRepo.RemoveVoteWithTransaction(reportID, userID)
	if err != nil {
		return nil, err
	}

	return &model.VoteResponse{
		VoteScore:    newScore,
		UserVoteType: nil,
	}, nil
}

// Returns user's current vote type and the report's total score.
func (s *VoteService) GetUserVote(reportIDStr, userIDStr string) (*model.VoteResponse, error) {
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid report ID")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	vote, err := s.voteRepo.GetVote(reportID, userID)
	if err != nil {
		return nil, err
	}

	score, err := s.voteRepo.GetReportVoteScore(reportID)
	if err != nil {
		return nil, err
	}

	response := &model.VoteResponse{
		VoteScore: score,
	}

	if vote != nil {
		response.UserVoteType = &vote.VoteType
	}

	return response, nil
}
