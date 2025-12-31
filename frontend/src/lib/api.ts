// API client for CityConnect backend

import type {
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  ReportListResponse,
  Report,
  CreateReportRequest,
  UpdateReportRequest,
  VoteRequest,
  VoteResponse,
  NotificationListResponse,
  CategoriesResponse,
  User,
  Category,
} from "@/types";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private getToken(): string | null {
    if (typeof window === "undefined") return null;
    return localStorage.getItem("token");
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const token = this.getToken();
    const headers: HeadersInit = {
      "Content-Type": "application/json",
      ...options.headers,
    };

    if (token) {
      (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response
        .json()
        .catch(() => ({ error: "Request failed" }));
      throw new Error(error.error || "Request failed");
    }

    return response.json();
  }

  // Auth endpoints
  async login(data: LoginRequest): Promise<LoginResponse> {
    return this.request<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async register(
    data: RegisterRequest
  ): Promise<{ message: string; user: User }> {
    return this.request("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async getMe(): Promise<User> {
    return this.request<User>("/api/v1/auth/me");
  }

  // Report endpoints
  async getPublicReports(
    search?: string,
    categoryId?: number | null
  ): Promise<ReportListResponse> {
    const params = new URLSearchParams();
    if (search) params.append("search", search);
    if (categoryId) params.append("category_id", categoryId.toString());
    const query = params.toString();
    return this.request<ReportListResponse>(
      `/api/v1/reports/public${query ? `?${query}` : ""}`
    );
  }

  async getMyReports(
    search?: string,
    categoryId?: number | null
  ): Promise<ReportListResponse> {
    const params = new URLSearchParams();
    if (search) params.append("search", search);
    if (categoryId) params.append("category_id", categoryId.toString());
    const query = params.toString();
    return this.request<ReportListResponse>(
      `/api/v1/reports/my${query ? `?${query}` : ""}`
    );
  }

  async getReports(): Promise<ReportListResponse> {
    return this.request<ReportListResponse>("/api/v1/reports/");
  }

  async getReport(id: string): Promise<Report> {
    return this.request<Report>(`/api/v1/reports/${id}`);
  }

  async createReport(
    data: CreateReportRequest
  ): Promise<{ message: string; report: Report }> {
    return this.request("/api/v1/reports/", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async updateReport(
    id: string,
    data: UpdateReportRequest
  ): Promise<{ message: string; report: Report }> {
    return this.request(`/api/v1/reports/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  }

  async updateReportStatus(
    id: string,
    status: string
  ): Promise<{ message: string }> {
    return this.request(`/api/v1/reports/${id}/status`, {
      method: "PATCH",
      body: JSON.stringify({ status }),
    });
  }

  async getCategories(): Promise<CategoriesResponse> {
    return this.request<CategoriesResponse>("/api/v1/reports/categories");
  }

  async createCategory(
    name: string,
    department: string
  ): Promise<{ message: string; category: Category }> {
    return this.request("/api/v1/reports/categories", {
      method: "POST",
      body: JSON.stringify({ name, department }),
    });
  }

  // Vote endpoints
  async castVote(
    reportId: string,
    voteType: VoteRequest
  ): Promise<VoteResponse> {
    return this.request<VoteResponse>(`/api/v1/reports/${reportId}/vote`, {
      method: "POST",
      body: JSON.stringify(voteType),
    });
  }

  async removeVote(reportId: string): Promise<VoteResponse> {
    return this.request<VoteResponse>(`/api/v1/reports/${reportId}/vote`, {
      method: "DELETE",
    });
  }

  async getVote(reportId: string): Promise<VoteResponse> {
    return this.request<VoteResponse>(`/api/v1/reports/${reportId}/vote`);
  }

  // Notification endpoints
  async getNotifications(): Promise<NotificationListResponse> {
    return this.request<NotificationListResponse>("/api/v1/notifications");
  }

  async markNotificationRead(
    notificationId: string
  ): Promise<{ message: string }> {
    return this.request(`/api/v1/notifications/${notificationId}/read`, {
      method: "PATCH",
    });
  }

  async markAllNotificationsRead(): Promise<{ message: string }> {
    return this.request("/api/v1/notifications/read-all", {
      method: "PATCH",
    });
  }

  // SSE stream URL - takes token parameter for external use
  getNotificationStreamUrl(token: string): string {
    return `${this.baseUrl}/api/v1/notifications/stream?token=${token}`;
  }
}

export const api = new ApiClient(API_BASE);
