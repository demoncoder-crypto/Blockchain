import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../store/authStore';
import { portfolioService, orderService } from '../services/api';
import { Balance, Order } from '../types'; // Import types
import useWebSocket from '../hooks/useWebSocket'; // Import the hook
import OrderForm from '../components/OrderForm'; // Import OrderForm
import OrderBook from '../components/OrderBook'; // Import OrderBook

// Define the structure for storing latest prices
interface LatestPrices {
  [symbol: string]: number;
}

// Construct WebSocket URL (adjust if different from API base)
// Note: Vite uses import.meta.env, not process.env
const WS_URL = (import.meta.env.VITE_WS_URL || 'ws://localhost:8080') + '/ws/prices';
console.log(`WebSocket URL: ${WS_URL}`);

function DashboardPage() {
  const { user, logout } = useAuthStore((state) => ({ user: state.user, logout: state.logout }));
  const navigate = useNavigate();

  const [balances, setBalances] = useState<Balance[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [latestPrices, setLatestPrices] = useState<LatestPrices>({});
  const [loadingPortfolio, setLoadingPortfolio] = useState(true);
  const [loadingOrders, setLoadingOrders] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedSymbol, setSelectedSymbol] = useState('BTC-USD'); // State for selected symbol

  // --- Function to fetch orders ---
  const fetchOrders = useCallback(async () => {
    setLoadingOrders(true);
    setError(null); // Clear previous errors
    try {
      const ordersRes = await orderService.getOrders();
      setOrders(ordersRes.data);
    } catch (err) {
      console.error('Failed to fetch orders:', err);
      setError('Failed to load orders.');
    } finally {
      setLoadingOrders(false);
    }
  }, []);

  // --- Initial Data Fetch (Portfolio + Orders) ---
  useEffect(() => {
    const fetchPortfolio = async () => {
      setLoadingPortfolio(true);
      setError(null); // Clear previous errors
      try {
        const portfolioRes = await portfolioService.getPortfolio();
        setBalances(portfolioRes.data);
      } catch (err) {
        console.error('Failed to fetch portfolio:', err);
        setError('Failed to load portfolio data.');
      } finally {
        setLoadingPortfolio(false);
      }
    };

    fetchPortfolio();
    fetchOrders(); // Initial fetch for orders
  }, [fetchOrders]); // Dependency array includes fetchOrders

  // --- WebSocket Handler ---
  const handlePriceUpdate = useCallback((data: { symbol: string; price: number }) => {
    setLatestPrices((prevPrices) => ({
      ...prevPrices,
      [data.symbol]: data.price,
    }));
  }, []);

  // --- WebSocket Connection ---
  const { isConnected } = useWebSocket({
    url: WS_URL,
    onMessage: handlePriceUpdate,
    onError: (event) => console.error('WebSocket Error:', event),
    onClose: (event) => console.log('WebSocket Closed:', event.code, event.reason),
  });

  // --- Logout Handler ---
  const handleLogout = () => {
    logout();
    navigate('/login'); // Redirect to login after logout
  };

  // --- Order Cancellation Handler ---
  const handleCancelOrder = async (orderId: string) => {
    console.log('Attempting to cancel order:', orderId);
    try {
        const response = await orderService.cancelOrder(orderId);
        console.log('Cancel response:', response.data);
        // Refresh orders list on successful cancellation
        fetchOrders(); 
    } catch(err: any) {
        console.error('Failed to cancel order:', orderId, err);
        if (err.response && err.response.data && err.response.data.error) {
            alert(`Failed to cancel order: ${err.response.data.error}`);
        } else {
            alert('Failed to cancel order: An unexpected error occurred.');
        }
    }
  };

  // Callback for OrderForm success
  const handleOrderSuccess = () => {
      fetchOrders(); // Refresh orders list
      // TODO: Optionally refresh portfolio as well, though it might take time for balance updates
      // fetchPortfolio();
  };

  return (
    <div>
      <h2>Dashboard</h2>
      {user && <p>Welcome, {user.username}!</p>}
      <button onClick={handleLogout}>Logout</button>
      <hr />

      {/* Display WebSocket Connection Status */}
      <p>Price Feed Status: {isConnected ? 'Connected' : 'Disconnected'}</p>
      <hr />

      {error && <p style={{ color: 'red' }}>{error}</p>}

      {/* --- Components --- */}
      <div style={{ display: 'flex', gap: '20px', flexWrap: 'wrap' }}> {/* Added flexWrap */}
        {/* Left Column: Portfolio & Orders */}
        <div style={{ flex: '1 1 400px' }}> {/* Added basis */}
          {/* Portfolio Section - TODO: Show value based on latestPrices */}
          <div>
            <h3>Portfolio Balances</h3>
            {loadingPortfolio ? (
              <p>Loading balances...</p>
            ) : balances.length > 0 ? (
              <table>
                <thead>
                  <tr>
                    <th>Asset</th>
                    <th>Available</th>
                    <th>Locked</th>
                    <th>Total</th>
                    {/* TODO: Add Value Column */} 
                  </tr>
                </thead>
                <tbody>
                  {balances.map((bal) => (
                    <tr key={bal.asset}>
                      <td>{bal.asset}</td>
                      <td>{bal.available.toFixed(8)}</td> {/* Adjust precision as needed */}
                      <td>{bal.locked.toFixed(8)}</td>
                      <td>{(bal.available + bal.locked).toFixed(8)}</td>
                      {/* <td>{( (bal.available + bal.locked) * (latestPrices[bal.asset+"-USD"] || 0) ).toFixed(2) }</td> */}
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <p>No balances found.</p>
            )}
          </div>
          <hr />
          {/* Open Orders Section */}
          <div>
            <h3>Open Orders</h3>
            <button onClick={fetchOrders} disabled={loadingOrders}>
              {loadingOrders ? 'Refreshing...' : 'Refresh Orders'}
            </button>
            {loadingOrders ? (
              <p>Loading orders...</p>
            ) : orders.length > 0 ? (
              <table>
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Symbol</th>
                    <th>Type</th>
                    <th>Side</th>
                    <th>Price</th>
                    <th>Quantity</th>
                    <th>Status</th>
                    <th>Created At</th>
                    <th>Action</th>
                  </tr>
                </thead>
                <tbody>
                  {orders.map((order) => (
                    <tr key={order.id}>
                      <td>{order.id.substring(0, 8)}...</td> {/* Shorten ID */}
                      <td>{order.symbol}</td>
                      <td>{order.type}</td>
                      <td>{order.side}</td>
                      <td>{order.price ? order.price.toFixed(2) : 'N/A'}</td> {/* Adjust precision */}
                      <td>{order.quantity.toFixed(8)}</td> {/* Adjust precision */}
                      <td>{order.status}</td>
                      <td>{new Date(order.created_at).toLocaleString()}</td>
                      <td>
                        {order.status === 'open' && (
                          <button onClick={() => handleCancelOrder(order.id)}>
                            Cancel
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <p>No open orders.</p>
            )}
          </div>
        </div>

        {/* Right Column: Tickers, Order Form, Order Book */}
        <div style={{ flex: '1 1 300px' }}> {/* Added basis */}
          {/* Price Tickers Section */}
          <div>
            <h3>Live Prices</h3>
            {Object.keys(latestPrices).length > 0 ? (
              <ul>
                {Object.entries(latestPrices).map(([symbol, price]) => (
                  <li key={symbol}>
                    {symbol}: {price.toFixed(2)} {/* Adjust precision */}
                  </li>
                ))}
              </ul>
            ) : (
              <p>{isConnected ? 'Waiting for price updates...' : 'Connecting...'}</p>
            )}
          </div>
          <hr />
          {/* Order Form Component */}
          <div>
            <h3>Place Order</h3>
            <OrderForm handleOrderSuccess={handleOrderSuccess} /> {/* Pass callback */}
          </div>
          <hr />
          {/* Order Book Component */}
          <div>
            <h3>Order Book ({selectedSymbol})</h3>
            <OrderBook symbol={selectedSymbol} /> {/* Render OrderBook */}
          </div>
        </div>
      </div>

    </div>
  );
}

export default DashboardPage; 