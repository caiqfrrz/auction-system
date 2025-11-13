export const BASE_URL = "http://localhost:8080";

export async function api(
  endpoint: string,
  options?: RequestInit
): Promise<any> {
  const url = `${BASE_URL}${endpoint}`;

  const response = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
    credentials: "include",
  });

  const text = await response.text();

  if (!response.ok) {
    try {
      const error = JSON.parse(text);
      throw new Error(error.error || error.message || `HTTP ${response.status}`);
    } catch {
      throw new Error(`HTTP ${response.status}: ${text}`);
    }
  }

  return text ? JSON.parse(text) : {};
}