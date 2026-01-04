export type Role =
  | "warga"
  | "admin_kebersihan"
  | "admin_kesehatan"
  | "admin_infrastruktur";
export type PrivacyLevel = "public" | "private" | "anonymous";
export type ReportStatus =
  | "pending"
  | "accepted"
  | "in_progress"
  | "completed"
  | "rejected";
export type VoteType = "upvote" | "downvote";

export interface User {
  id: string;
  email: string;
  name: string;
  role: Role;
  department?: string;
}

export interface Category {
  id: number;
  name: string;
  department: string;
}

export interface Report {
  id: string;
  title: string;
  description: string;
  category_id: number;
  category?: Category;
  location_lat?: number;
  location_lng?: number;
  photo_url?: string;
  privacy_level: PrivacyLevel;
  reporter_id?: string;
  reporter_name?: string;
  status: ReportStatus;
  vote_score: number;
  created_at: string;
  updated_at: string;
}

export interface Notification {
  id: string;
  user_id: string;
  report_id?: string;
  title: string;
  message: string;
  is_read: boolean;
  created_at: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  name: string;
  role: Role;
}

export interface CreateReportRequest {
  title: string;
  description: string;
  category_id: number;
  location_lat?: number;
  location_lng?: number;
  privacy_level: PrivacyLevel;
}

export interface UpdateReportRequest {
  title?: string;
  description?: string;
}

export interface VoteRequest {
  vote_type: VoteType;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface ReportListResponse {
  reports: Report[];
  total: number;
}

export interface VoteResponse {
  vote_score: number;
  user_vote_type?: VoteType;
}

export interface NotificationListResponse {
  notifications: Notification[];
  unread_count: number;
}

export interface CategoriesResponse {
  categories: Category[];
}
