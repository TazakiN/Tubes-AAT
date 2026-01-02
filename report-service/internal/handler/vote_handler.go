package handler

import (
	"net/http"

	"report-service/internal/model"
	"report-service/internal/service"

	"github.com/gin-gonic/gin"
)

type VoteHandler struct {
	voteService *service.VoteService
}

func NewVoteHandler(voteService *service.VoteService) *VoteHandler {
	return &VoteHandler{voteService: voteService}
}

func (h *VoteHandler) CastVote(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	reportID := c.Param("id")
	if reportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "report ID required"})
		return
	}

	var req model.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate vote type
	if req.VoteType != model.VoteUpvote && req.VoteType != model.VoteDownvote {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vote_type must be 'upvote' or 'downvote'"})
		return
	}

	response, err := h.voteService.CastVote(reportID, userID, req.VoteType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *VoteHandler) RemoveVote(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	reportID := c.Param("id")
	if reportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "report ID required"})
		return
	}

	response, err := h.voteService.RemoveVote(reportID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *VoteHandler) GetVote(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	reportID := c.Param("id")
	if reportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "report ID required"})
		return
	}

	response, err := h.voteService.GetUserVote(reportID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
