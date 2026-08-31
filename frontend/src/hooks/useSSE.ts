import { useEffect, useRef } from "react";
import { useUser } from "./useUser";
import type { Notification } from "../lib/types";
import { BASE_URL } from "../lib/api";

export function useSSE(
    registeredAuctionsID: string[],
    onNotification: (notification: Notification) => void
) {
    const eventSourcesRef = useRef<Map<string, EventSource>>(new Map());
    const { userId } = useUser();

    useEffect(() => {
        
        registeredAuctionsID.forEach((auctionId) => {
        if (!eventSourcesRef.current.has(auctionId)) {
            const eventSource = new EventSource(
            `${BASE_URL}/register-interest/${auctionId}/stream?clienteID=${userId}`
            );

            eventSource.addEventListener('lance_validado', (e) => {
                const data = JSON.parse(e.data);
                onNotification({ ...data, auctionId });
            });

            eventSource.addEventListener('lance_invalidado', (e) => {
                const data = JSON.parse(e.data);
                onNotification({ ...data, auctionId });
            });

            eventSource.addEventListener('leilao_vencedor', (e) => {
                const data = JSON.parse(e.data);
                onNotification({ ...data, auctionId });
            });

            eventSource.addEventListener('link_pagamento', (e) => {
                const data = JSON.parse(e.data);
                onNotification({ ...data, auctionId });
            });

            eventSource.addEventListener('status_pagamento', (e) => {
                const data = JSON.parse(e.data);
                onNotification({ ...data, auctionId });
            });

            eventSource.onerror = (error) => {
                console.error(`SSE error for auction ${auctionId}:`, error);
                eventSource.close();
                eventSourcesRef.current.delete(auctionId);
            };

            eventSourcesRef.current.set(auctionId, eventSource);
            console.log(`Connected to auction ${auctionId}`);
        }
        });

        eventSourcesRef.current.forEach((eventSource, auctionId) => {
        if (!registeredAuctionsID.includes(auctionId)) {
            eventSource.close();
            eventSourcesRef.current.delete(auctionId);
            console.log(`Disconnected from auction ${auctionId}`);
        }
        });

        return () => {
            eventSourcesRef.current.forEach((eventSource) => {
                eventSource.close();
            });
            eventSourcesRef.current.clear();
        };

    }, [registeredAuctionsID, userId]);
}