"use client";

import type { Report, VoteType } from "@/types";
import { useState } from "react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

interface ReportCardProps {
  report: Report;
  showVoting?: boolean;
  onStatusChange?: () => void;
}

export default function ReportCard({
  report,
  showVoting = true,
  onStatusChange,
}: ReportCardProps) {
  const { user } = useAuth();
  const [voteScore, setVoteScore] = useState(report.vote_score);
  const [userVote, setUserVote] = useState<VoteType | null>(null);
  const [isVoting, setIsVoting] = useState(false);

  const handleVote = async (voteType: VoteType) => {
    if (!user || isVoting) return;

    setIsVoting(true);
    try {
      if (userVote === voteType) {
        // Remove vote
        const response = await api.removeVote(report.id);
        setVoteScore(response.vote_score);
        setUserVote(null);
      } else {
        // Cast vote
        const response = await api.castVote(report.id, { vote_type: voteType });
        setVoteScore(response.vote_score);
        setUserVote(response.user_vote_type || null);
      }
    } catch (error) {
      console.error("Vote failed:", error);
    } finally {
      setIsVoting(false);
    }
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString("id-ID", {
      day: "numeric",
      month: "short",
      year: "numeric",
    });
  };

  return (
    <div className="card report-card">
      {showVoting && report.privacy_level === "public" && (
        <div className="vote-container">
          <button
            className={`vote-btn ${userVote === "upvote" ? "active-up" : ""}`}
            onClick={() => handleVote("upvote")}
            disabled={isVoting || !user}
            title={user ? "Upvote" : "Login untuk vote"}
          >
            ▲
          </button>
          <span className="vote-score">{voteScore}</span>
          <button
            className={`vote-btn ${
              userVote === "downvote" ? "active-down" : ""
            }`}
            onClick={() => handleVote("downvote")}
            disabled={isVoting || !user}
            title={user ? "Downvote" : "Login untuk vote"}
          >
            ▼
          </button>
        </div>
      )}

      <h3
        className="card-title"
        style={{ paddingRight: showVoting ? "60px" : "0" }}
      >
        {report.title}
      </h3>

      <p className="card-description">
        {report.description.length > 150
          ? report.description.substring(0, 150) + "..."
          : report.description}
      </p>

      <div className="report-meta">
        <span className={`status-badge status-${report.status}`}>
          {report.status.replace("_", " ")}
        </span>

        {report.category && (
          <span className="category-badge">{report.category.name}</span>
        )}

        <span className={`privacy-badge privacy-${report.privacy_level}`}>
          {report.privacy_level}
        </span>

        <span className="report-meta-item">
          📅 {formatDate(report.created_at)}
        </span>

        {report.reporter_name && (
          <span className="report-meta-item">👤 {report.reporter_name}</span>
        )}
      </div>
    </div>
  );
}
