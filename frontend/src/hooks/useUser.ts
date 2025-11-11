import { useState } from "react";


export function useUser() {
    const [userId, setUserId] = useState<string>(() => {
        const stored = localStorage.getItem("userId");

        if (stored) return stored;

        const newId = crypto.randomUUID();
        localStorage.setItem("userId", newId);
        return newId
    })

    const clearUser = () => {
        localStorage.removeItem("userId");
        const newId = crypto.randomUUID();
        setUserId(newId);
        localStorage.setItem("userId", newId);
    }

    return {userId, clearUser};
}