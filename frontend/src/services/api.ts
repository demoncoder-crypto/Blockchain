import axios from 'axios';
import { useAuthStore } from '../store/authStore'; // Import the store

// Determine base URL based on environment
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

console.log(`API Base URL: ${API_BASE_URL}`); // For debugging

// Create an Axios instance
const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// --- Request Interceptor (for adding JWT token) ---
apiClient.interceptors.request.use(
  (config) => {
    // Get token from Zustand store
    const token = useAuthStore.getState().token;
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// --- Response Interceptor (for global error handling, e.g., 401)
apiClient.interceptors.response.use(
  (response) => response, // Pass through successful responses
  (error) => {
    if (error.response && error.response.status === 401) {
      console.error('Unauthorized access - 401', error.response);
      // Trigger logout action from Zustand store
      useAuthStore.getState().logout();
      // Redirect logic might be better handled in a component effect listening to auth state
      // window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// --- Define API Service Functions ---

// Example: Auth Service
export const authService = {
  login: (credentials: { username: string; password: string }) =>
    apiClient.post('/auth/login', credentials),

  signup: (userData: { username: string; password: string }) =>
    apiClient.post('/auth/signup', userData),

  // Add other auth-related calls if needed (e.g., fetch user profile /me)
  getProfile: () => apiClient.get('/me'),
};

// Example: Order Service
export const orderService = {
  // Define types for order creation request and response if different from backend models
  createOrder: (orderData: any) => apiClient.post('/orders', orderData),
  getOrders: () => apiClient.get('/orders'),
  getOrderById: (id: string) => apiClient.get(`/orders/${id}`),
  cancelOrder: (id: string) => apiClient.delete(`/orders/${id}`),
};

// Example: Portfolio Service
export const portfolioService = {
  getPortfolio: () => apiClient.get('/portfolio'),
};

// Example: Market Data Service (Order Book)
export const marketService = {
  getOrderBook: (symbol: string) => apiClient.get(`/book/${symbol}`),
};

export default apiClient; 