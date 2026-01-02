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

type VoteService struct {
	voteRepo   *repository.VoteRepository
	reportRepo *repository.ReportRepository
	rmq        *messaging.RabbitMQ
}

func NewVoteService(voteRepo *repository.VoteRepository, reportRepo *repository.ReportRepository, rmq *messaging.RabbitMQ) *VoteService {
	return &VoteService{
		voteRepo:   voteRepo,
		reportRepo: reportRepo,
		rmq:        rmq,
	}
}

func (s *VoteService) CastVote(reportIDStr, userIDStr string, voteType model.VoteType) (*model.VoteResponse, error) {
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid report ID")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	if voteType != model.VoteUpvote && voteType != model.VoteDownvote {
		return nil, fmt.Errorf("invalid vote type")
	}

	// hanya report publik yang bisa di-vote
	report, err := s.reportRepo.FindByID(reportID)
	if err != nil {
		return nil, fmt.Errorf("report not found")
	}

	if report.PrivacyLevel != model.PrivacyPublic {
		return nil, fmt.Errorf("can only vote on public reports")
	}

	newScore, err := s.voteRepo.VoteWithTransaction(reportID, userID, voteType)
	if err != nil {
		return nil, err
	}

	// async publish ke RabbitMQ
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
				log.Printf("rmq publish failed: %v", err)
			}
		}()
	}

	return &model.VoteResponse{
		VoteScore:    newScore,
		UserVoteType: &voteType,
	}, nil
}

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
