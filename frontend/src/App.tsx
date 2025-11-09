import { useEffect, useState } from "react";
import "./App.css";
import type { Auction } from "./lib/types";
import { api } from "./lib/api";

function App() {
  const [auctions, setAuctions] = useState<Auction[]>([]);

  useEffect(() => {
    const fetchAuctions = async () => {
      try {
        const auctions = await api("/consult-auctions");

        // let auctionTyped: Auction[] = [];
        // for (const a of auctions) {
        //   auctionTyped = [
        //     ...auctionTyped,
        //     {
        //       id: a.id,
        //       description: a.description,
        //       start: a.start,
        //       end: a.end,
        //     },
        //   ];
        // }

        // setAuctions(auctions);
      } catch (error) {
        console.error("error fetching auctions:", error);
      }
    };

    fetchAuctions();
  }, []);

  return (
    <>
      <div className="header">
        <h1 className="title">Auction System</h1>
      </div>

      <h3>Current auctions:</h3>
      <div className="auction-container">
        {auctions.map((a) => (
          <div>
            <p>{a.description}</p>
            <p>{a.id}</p>
            <p>{a.start.getDate()}</p>
            <p>{a.end.getDate()}</p>
          </div>
        ))}
      </div>
    </>
  );
}

export default App;
