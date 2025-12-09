import { useEffect, useState } from "react";
import "./App.css";
import AuctionView from "./AuctionView";
import type { Auction } from "./lib/types";
import { api } from "./lib/api";
import bellIcon from "./assets/bell.svg";
import selectedBellIcon from "./assets/bell-selected.svg";
import { useSSE } from "./hooks/useSSE";
import toast, { Toaster } from "react-hot-toast";
import { useUser } from "./hooks/useUser";

const REFRESH_AUCTIONS_TIME = 5 * 1000; // 5 seconds

function App() {
  const [auctions, setAuctions] = useState<Auction[]>([]);
  const [showCreateAuction, setShowCreateAuction] = useState<boolean>(false);
  const [showAuctionDetails, setShowAuctionDetails] = useState<boolean>(false);
  const [auctionDetails, setAuctionDetails] = useState<Auction | null>(null);
  const [registeredAuctionsID, setRegisteredAuctionsID] = useState<string[]>(
    []
  );
  const [formData, setFormData] = useState({
    description: "",
    start: "",
    end: "",
  });
  const { userId } = useUser();

  const handleNotification = (notification: any) => {
    const { type, data, leilao_id } = notification;
    const auction = auctions.find((a: Auction) => a.id === String(leilao_id));

    switch (type) {
      case "lance_validado":
        toast.success(
          data.user_id === userId
            ? `Você fez um novo lance de R$ ${data.valor} no leilão ${auction?.description}!`
            : `Novo lance de R$ ${data.valor} no leilão ${auction?.description}!`,
          {
            duration: 4000,
            icon: "🔨",
          }
        );

        if (auctionDetails?.id === leilao_id) {
          // fetchAuctionBids(auctionId);
        }
        break;

      case "lance_invalidado":
        toast.error(`Lance inválido: ${data.motivo}`, {
          duration: 5000,
          icon: "❌",
        });
        break;

      case "leilao_vencedor":
        if (data.vencedor_id === userId) {
          toast.success("🎉 Você venceu o leilão!", {
            duration: 6000,
          });
        } else {
          toast(
            `Leilão ${leilao_id} encerrado por R$${data.valor_final}. ID do Vencedor: ${data.vencedor_id}`,
            {
              duration: 5000,
              icon: "ℹ️",
            }
          );
        }
        break;

      case "link_pagamento":
        toast(
          (t) => (
            <div>
              <p>Pagamento disponível!</p>
              <button
                onClick={() => {
                  window.open(data.payment_link, "_blank");
                  toast.dismiss(t.id);
                }}
              >
                Ir para pagamento
              </button>
            </div>
          ),
          {
            duration: Infinity,
          }
        );
        break;

      case "status_pagamento":
        if (data.status === "approved") {
          toast.success("✅ Pagamento aprovado!", {
            duration: 20000,
          });
        } else {
          toast.error("❌ Pagamento recusado", {
            duration: 20000,
          });
        }
        break;
    }
  };

  useSSE(registeredAuctionsID, handleNotification);

  const onPressBell = (auctionId: string) => {
    if (auctionId == "") return;

    if (registeredAuctionsID.includes(auctionId)) {
      setRegisteredAuctionsID(
        registeredAuctionsID.filter((id: string) => id != auctionId)
      );
    } else {
      setRegisteredAuctionsID([...registeredAuctionsID, auctionId]);
    }
  };

  const fetchAuctions = async () => {
    try {
      const response = await api("/consult-auctions");

      const auctionsTyped: Auction[] = response.auctions.map((a: any) => ({
        id: a.id,
        description: a.description,
        start: new Date(a.start),
        end: new Date(a.end),
        active: a.active,
      }));

      setAuctions(auctionsTyped);
    } catch (error) {
      console.error("error fetching auctions:", error);
    }
  };

  useEffect(() => {
    fetchAuctions();

    const interval = setInterval(() => {
      fetchAuctions();
    }, REFRESH_AUCTIONS_TIME);

    return () => {
      clearInterval(interval);
    };
  }, []);

  const submitCreateAuction = async (e: React.FormEvent) => {
    e.preventDefault();

    try {
      const payload = {
        description: formData.description,
        start: new Date(formData.start).toISOString(),
        end: new Date(formData.end).toISOString(),
      };

      const response = await api("/create-auction", {
        method: "POST",
        body: JSON.stringify(payload),
      });

      setShowCreateAuction(false);
      setFormData({
        description: "",
        start: "",
        end: "",
      });

      await fetchAuctions();
    } catch (error) {
      console.error("error creating auction:", error);
      alert(
        `Failed to create auction: ${
          error instanceof Error ? error.message : "Unknown error"
        }`
      );
    }
  };

  return (
    <>
      <Toaster position="top-right" />
      <div className="header">
        <h1 className="title">Auction System</h1>
      </div>

      <h3>Current auctions:</h3>
      <div className="auction-container">
        {auctions.map((a) => (
          <div
            key={a.id}
            className="auction-item"
            onClick={() => {
              setAuctionDetails(a);
              setShowAuctionDetails(true);
            }}
          >
            {a.active && (
              <button
                className="bell-button"
                onClick={(e) => {
                  e.stopPropagation();
                  onPressBell(a.id);
                }}
              >
                <img
                  src={
                    registeredAuctionsID.includes(a.id)
                      ? selectedBellIcon
                      : bellIcon
                  }
                  width={20}
                  height={20}
                />
              </button>
            )}
            <p className="description-auction">
              <strong>{a.description}</strong>
            </p>
            <p>Start: {a.start.toLocaleString()}</p>
            <p>End: {a.end.toLocaleString()}</p>
            {a.active ? (
              <p style={{ color: "green" }}>Active!</p>
            ) : (
              <p style={{ color: "red" }}>{"Not active :("}</p>
            )}
          </div>
        ))}
        <button
          className="auction-item"
          onClick={() => setShowCreateAuction(!showCreateAuction)}
        >
          <p>Create new auction</p>
        </button>
      </div>
      {showCreateAuction && (
        <div
          className="modal-overlay"
          onClick={() => setShowCreateAuction(false)}
        >
          <div className="modal-auction" onClick={(e) => e.stopPropagation()}>
            {/* Conteúdo do modal aqui */}
            <h2>Create New Auction</h2>
            <form className="auction-form" onSubmit={submitCreateAuction}>
              <div className="auction-form-field">
                <label>Description:</label>
                <input
                  type="text"
                  value={formData.description}
                  onChange={(e) =>
                    setFormData({ ...formData, description: e.target.value })
                  }
                  required
                />
              </div>
              <div className="auction-form-field">
                <label>Start Date:</label>
                <input
                  type="datetime-local"
                  value={formData.start}
                  onChange={(e) =>
                    setFormData({ ...formData, start: e.target.value })
                  }
                  required
                />
              </div>
              <div className="auction-form-field">
                <label>End Date:</label>
                <input
                  type="datetime-local"
                  value={formData.end}
                  onChange={(e) =>
                    setFormData({ ...formData, end: e.target.value })
                  }
                  required
                />
              </div>
              <div className="button-container">
                <button type="submit" className="create-button">
                  Create Auction
                </button>
                <button
                  type="button"
                  className="cancel-button"
                  onClick={() => {
                    setShowCreateAuction(false);
                    setFormData({
                      description: "",
                      start: "",
                      end: "",
                    });
                  }}
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
      {showAuctionDetails && (
        <AuctionView
          onClose={() => setShowAuctionDetails(false)}
          auction={auctionDetails}
        />
      )}
    </>
  );
}

export default App;
