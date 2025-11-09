const BASE_URL = "http://127.0.0.1:8080"

export const api = async (
    path: string,
    options: RequestInit = {},
): Promise<any> => {
    const headers: Record<string, string> = {
        "Content-Type": "application/json",
        ...(options.headers as Record<string, string> || {})
    };

    const res = await fetch(`${BASE_URL}${path}`, {
        ...options,
        headers,
    });

    const json = await res.json();

    if (!res.ok) {
      const errorMessage = json.message || json.error || "Unknown error";
      const error = new Error(errorMessage);

      if (json.error_code) {
          (error as any).error_code = json.error_code;
      }

      (error as any).response = json;
      throw error;
    }

    return json;
}