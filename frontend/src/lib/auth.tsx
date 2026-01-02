"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";
import type { User } from "@/types";
import { api } from "./api";

interface AuthContextType {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (
    email: string,
    password: string,
    name: string,
    role: string
  ) => Promise<void>;
  logout: () => void;
  isAdmin: () => boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isMounted, setIsMounted] = useState(false);

  useEffect(() => {
    setIsMounted(true);
  }, []);

  useEffect(() => {
    if (!isMounted) return;

    const storedToken = localStorage.getItem("token");
    if (storedToken) {
      setToken(storedToken);
      api
        .getMe()
        .then(setUser)
        .catch(() => {
          localStorage.removeItem("token");
          setToken(null);
        })
        .finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, [isMounted]);

  const login = async (email: string, password: string) => {
    const response = await api.login({ email, password });
    localStorage.setItem("token", response.token);
    setToken(response.token);
    setUser(response.user);
  };

  const register = async (
    email: string,
    password: string,
    name: string,
    role: string
  ) => {
    await api.register({
      email,
      password,
      name,
      role: role as
        | "warga"
        | "admin_kebersihan"
        | "admin_kesehatan"
        | "admin_infrastruktur",
    });
  };

  const logout = () => {
    localStorage.removeItem("token");
    setToken(null);
    setUser(null);
  };

  const isAdmin = () => {
    return user?.role?.startsWith("admin_") ?? false;
  };

  return (
    <AuthContext.Provider
      value={{ user, token, isLoading, login, register, logout, isAdmin }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
