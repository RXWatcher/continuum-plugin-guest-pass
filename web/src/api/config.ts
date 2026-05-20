import { api } from "@/lib/api";
import type { AppConfig } from "@/lib/types";

export async function getConfig(): Promise<AppConfig> {
  return api<AppConfig>("/api/admin/config");
}

export async function updateConfig(cfg: AppConfig): Promise<AppConfig> {
  return api<AppConfig>("/api/admin/config", {
    method: "PATCH",
    body: JSON.stringify(cfg),
  });
}
