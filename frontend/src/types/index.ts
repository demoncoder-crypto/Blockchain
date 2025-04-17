// Based on backend/internal/models

export interface Balance {
  user_id: string; // Assuming UUID is string on frontend
  asset: string;
  available: number; // Go float64 maps to number
  locked: number;
  updated_at: string; // Assuming timestamp comes as string
}

export interface Order {
  id: string;
  user_id: string;
  symbol: string;
  type: 'limit' | 'market';
  side: 'buy' | 'sell';
  price?: number; // Optional for market orders
  quantity: number;
  status: 'open' | 'filled' | 'partially_filled' | 'cancelled' | 'pending';
  created_at: string;
  updated_at: string;
}

export interface UserProfile {
    user_id: string;
    username: string;
    // Add other fields from /me endpoint if needed
}

// Add other shared types as needed (e.g., PriceUpdate, OrderBookDepth) 