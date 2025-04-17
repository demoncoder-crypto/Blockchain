import React, { useState } from 'react';
import { orderService } from '../services/api';

// Define props including the success callback
interface OrderFormProps {
    handleOrderSuccess: () => void;
}

const OrderForm: React.FC<OrderFormProps> = ({ handleOrderSuccess }) => {
  const [symbol, setSymbol] = useState('BTC-USD'); // Default or selected symbol
  const [type, setType] = useState<'limit' | 'market'>('limit');
  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [price, setPrice] = useState(''); // Store as string for input control
  const [quantity, setQuantity] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setLoading(true);

    const orderData = {
      symbol,
      type,
      side,
      quantity: parseFloat(quantity),
      price: type === 'limit' ? parseFloat(price) : 0, // Price only relevant for limit
    };

    // Basic validation (more can be added)
    if (!orderData.symbol || !orderData.quantity || orderData.quantity <= 0) {
        setError('Please enter a valid symbol and positive quantity.');
        setLoading(false);
        return;
    }
    if (orderData.type === 'limit' && (!orderData.price || orderData.price <= 0)) {
        setError('Please enter a valid positive price for limit orders.');
        setLoading(false);
        return;
    }
    if (orderData.type === 'market') {
        // Backend currently doesn't support market orders fully
        // setError('Market orders are not yet supported.');
        // setLoading(false);
        // return;
        // For now, let it pass but backend will likely reject or handle partially
    }

    try {
      const response = await orderService.createOrder(orderData);
      console.log('Order created:', response.data);
      setSuccess(`Order ${response.data.id.substring(0,8)} created successfully!`);
      // Clear form on success
      setPrice('');
      setQuantity('');
      // Call the callback prop to notify parent (DashboardPage)
      handleOrderSuccess();
    } catch (err: any) {
      console.error('Order creation error:', err);
      if (err.response && err.response.data && err.response.data.error) {
        setError(`Order failed: ${err.response.data.error}`);
      } else {
        setError('Order failed: An unexpected error occurred.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      {/* Symbol Selection (Could be a dropdown) */}
      <div>
        <label htmlFor="symbol">Symbol:</label>
        <input 
          type="text" 
          id="symbol" 
          value={symbol} 
          onChange={(e) => setSymbol(e.target.value.toUpperCase())} 
          required 
          disabled={loading}
        />
      </div>

      {/* Order Type */} 
      <div>
        <label>Type:</label>
        <select value={type} onChange={(e) => setType(e.target.value as 'limit' | 'market')} disabled={loading}>
          <option value="limit">Limit</option>
          {/* <option value="market">Market</option> */}
        </select>
      </div>

       {/* Side */} 
       <div>
        <label>Side:</label>
        <select value={side} onChange={(e) => setSide(e.target.value as 'buy' | 'sell')} disabled={loading}>
          <option value="buy">Buy</option>
          <option value="sell">Sell</option>
        </select>
      </div>

      {/* Price (Only for Limit Orders) */} 
      {type === 'limit' && (
        <div>
          <label htmlFor="price">Price:</label>
          <input 
            type="number" 
            id="price" 
            value={price} 
            onChange={(e) => setPrice(e.target.value)} 
            required 
            min="0.00000001" // Example min value
            step="any" // Allow decimals
            disabled={loading}
          />
        </div>
      )}

      {/* Quantity */} 
      <div>
        <label htmlFor="quantity">Quantity:</label>
        <input 
          type="number" 
          id="quantity" 
          value={quantity} 
          onChange={(e) => setQuantity(e.target.value)} 
          required
          min="0.00000001" // Example min value
          step="any"
          disabled={loading}
        />
      </div>

      {error && <p style={{ color: 'red' }}>{error}</p>}
      {success && <p style={{ color: 'green' }}>{success}</p>}

      <button type="submit" disabled={loading}>
        {loading ? 'Placing Order...' : `Place ${side.toUpperCase()} Order`}
      </button>
    </form>
  );
};

export default OrderForm; 