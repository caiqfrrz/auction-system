export const BASE_URL = "http://127.0.0.1:8080"

export const api = async (
    path: string,
    options: RequestInit = {},
): Promise<any> => {
    const headers: Record<string, string> = {
        "Content-Type": "application/json",
        ...(options.headers as Record<string, string> || {})
    };

    try {
        const res = await fetch(`${BASE_URL}${path}`, {
            ...options,
            headers,
        });

        let json;
        try {
            json = await res.json();
        } catch (e) {
            const text = await res.text();
            json = { error: text || "Invalid response from server" };
        }

        if (!res.ok) {
            const errorMessage = json.error || json.message || json.details || `HTTP ${res.status}`;
            const errorDetails = json.details || "";
            
            console.error("API Error:", {
                status: res.status,
                path,
                body: json,
            });
            
            throw new Error(errorDetails ? `${errorMessage}: ${errorDetails}` : errorMessage);
        }

        return json;
    } catch (error) {
        if (error instanceof Error) {
            throw error;
        }
        throw new Error("Network error or invalid response");
    }
}