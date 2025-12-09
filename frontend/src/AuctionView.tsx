import { useEffect, useState } from "react";
import "./AuctionView.css";
import type { Auction } from "./lib/types";
import { api } from "./lib/api";
import { useUser } from "./hooks/useUser";

interface Props {
  auction: Auction | null;
  onClose: () => void;
}

function AuctionView({ auction, onClose }: Props) {
  if (auction === null) return null;

  const [bidValue, setBidValue] = useState<string>("");
  const [highestBid, setHighestBid] = useState<string>("");
  const { userId } = useUser();

  useEffect(() => {
    const getHighestBid = async () => {
      try {
        const data: any = await api(`/highest-bid?auctionId=${auction.id}`);
        if (data.highest_bid && data.highest_bid !== "") {
          setHighestBid(data.highest_bid);
        }
      } catch (error) {
        console.error("error fetching highest bid:", error);
      }
    };

    getHighestBid();
  }, [auction]);

  const submitBid = async (e: React.FormEvent) => {
    e.preventDefault();

    try {
      await api("/make-bid", {
        method: "POST",
        body: JSON.stringify({
          valor: Number(bidValue),
          leilao_id: auction.id,
          user_id: userId,
        }),
      });
    } catch (error) {
      console.error("error sending bid:", error);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-auction" onClick={(e) => e.stopPropagation()}>
        <h1>{auction.description}</h1>
        <p>Start time: {auction.start.toLocaleString()}</p>
        <p>End time: {auction.end.toLocaleString()}</p>
        {auction.active ? (
          <p style={{ color: "green" }}>Active!</p>
        ) : (
          <p style={{ color: "red" }}>{"Not active :("}</p>
        )}

        {auction.active && (
          <>
            <h3>Maior lance: {highestBid}</h3>
            <form className="form-bid" onSubmit={submitBid}>
              <h3>Make a bid:</h3>
              <input
                type="number"
                value={bidValue}
                onChange={(e) => setBidValue(e.target.value)}
                placeholder="Enter bid amount"
                required
              />
              <button type="submit">Place Bid</button>
            </form>
          </>
        )}
      </div>
    </div>
  );
}

export default AuctionView;
