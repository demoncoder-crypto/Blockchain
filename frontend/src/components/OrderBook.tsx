import React, { useState, useEffect } from 'react';
import { marketService } from '../services/api';

// Define types based on backend/internal/orderbook
interface BookLevel {
    price: number;
    quantity: number;
}
interface OrderBookData {
    symbol: string;
    bids: BookLevel[];
    asks: BookLevel[];
}

interface OrderBookProps {
    symbol: string; // Symbol to display
}

const OrderBook: React.FC<OrderBookProps> = ({ symbol }) => {
    const [depth, setDepth] = useState<OrderBookData | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchDepth = async () => {
            if (!symbol) return; // Don't fetch if no symbol
            setLoading(true);
            setError(null);
            try {
                const response = await marketService.getOrderBook(symbol);
                setDepth(response.data);
            } catch (err: any) {
                console.error(`Failed to fetch order book for ${symbol}:`, err);
                setError(`Failed to load order book for ${symbol}.`);
            } finally {
                setLoading(false);
            }
        };

        fetchDepth();

        // Optional: Set up polling or WebSocket for real-time updates
        const intervalId = setInterval(fetchDepth, 5000); // Refresh every 5 seconds (example)

        return () => clearInterval(intervalId); // Cleanup interval on unmount/symbol change

    }, [symbol]);

    if (loading) {
        return <p>Loading order book for {symbol}...</p>;
    }

    if (error) {
        return <p style={{ color: 'red' }}>{error}</p>;
    }

    if (!depth || (depth.bids.length === 0 && depth.asks.length === 0)) {
        return <p>Order book for {symbol} is empty.</p>;
    }

    // Simple table display
    return (
        <div>
            {/* Consider fixed height and overflow for long books */}
            <table>
                <thead>
                    <tr>
                        <th>Price (Asks)</th>
                        <th>Quantity</th>
                        <th>|</th>
                        <th>Price (Bids)</th>
                        <th>Quantity</th>
                    </tr>
                </thead>
                <tbody>
                    {/* Display Asks (lowest first, so reverse maybe?) */} 
                    {depth.asks.slice().reverse().slice(0, 10).map((ask, index) => ( // Show top 10 asks reversed
                        <tr key={`ask-${index}`}>
                            <td>{ask.price.toFixed(2)}</td>
                            <td>{ask.quantity.toFixed(8)}</td>
                            <td>|</td>
                            <td>-</td>
                            <td>-</td>
                        </tr>
                    ))}
                     <tr><td colSpan={5} style={{textAlign: 'center', fontWeight: 'bold'}}>--- Spread ---</td></tr> 
                    {/* Display Bids (highest first) */} 
                    {depth.bids.slice(0, 10).map((bid, index) => ( // Show top 10 bids
                         <tr key={`bid-${index}`}>
                            <td>-</td>
                            <td>-</td>
                            <td>|</td>
                            <td>{bid.price.toFixed(2)}</td>
                            <td>{bid.quantity.toFixed(8)}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
};

export default OrderBook; 