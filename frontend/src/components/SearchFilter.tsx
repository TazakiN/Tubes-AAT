"use client";

import { useState, useEffect } from "react";
import type { Category } from "@/types";

interface SearchFilterProps {
  categories: Category[];
  onSearch: (search: string, categoryId: number | null) => void;
  placeholder?: string;
}

export default function SearchFilter({
  categories,
  onSearch,
  placeholder = "Cari laporan...",
}: SearchFilterProps) {
  const [search, setSearch] = useState("");
  const [categoryId, setCategoryId] = useState<number | null>(null);

  useEffect(() => {
    // Debounce search
    const timer = setTimeout(() => {
      onSearch(search, categoryId);
    }, 300);
    return () => clearTimeout(timer);
  }, [search, categoryId, onSearch]);

  return (
    <div
      style={{
        display: "flex",
        gap: "1rem",
        marginBottom: "1.5rem",
        flexWrap: "wrap",
      }}
    >
      <div style={{ flex: 1, minWidth: "200px" }}>
        <input
          type="text"
          className="form-input"
          placeholder={placeholder}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={{ width: "100%" }}
        />
      </div>
      <div style={{ minWidth: "200px" }}>
        <select
          className="form-select"
          value={categoryId ?? ""}
          onChange={(e) =>
            setCategoryId(e.target.value ? parseInt(e.target.value) : null)
          }
          style={{ width: "100%" }}
        >
          <option value="">Semua Kategori</option>
          {categories.map((cat) => (
            <option key={cat.id} value={cat.id}>
              {cat.name} ({cat.department})
            </option>
          ))}
        </select>
      </div>
    </div>
  );
}
