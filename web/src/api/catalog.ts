import { api } from "@/lib/api";
import type { CatalogSearchResponse } from "@/lib/types";

export type CatalogSearchParams = {
  query: string;
  type?: "all" | "movie" | "episode";
  limit?: number;
  signal?: AbortSignal;
};

export async function searchCatalog({ query, type, limit, signal }: CatalogSearchParams): Promise<CatalogSearchResponse> {
  const params = new URLSearchParams({ q: query });
  if (limit) params.set("limit", String(limit));
  if (type && type !== "all") params.set("type", type);
  return api<CatalogSearchResponse>(`/api/admin/catalog/search?${params.toString()}`, { signal });
}
