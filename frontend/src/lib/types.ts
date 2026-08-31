

export type Auction = {
    id: string;
    description: string;
    start: Date;
    end: Date;
    active: boolean;
}

export interface Notification {
  type: 'lance_validado' | 'lance_invalidado' | 'leilao_vencedor' | 'link_pagamento' | 'status_pagamento';
  leilao_id: number;
  cliente_id?: number;
  data: any;
  timestamp: string;
  auctionId: string;
}