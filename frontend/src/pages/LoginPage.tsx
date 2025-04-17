import React, { useState, useEffect } from 'react';
import { useNavigate, Navigate } from 'react-router-dom';
import { useAuthStore, useIsAuthenticated } from '../store/authStore';
import { authService } from '../services/api';

function LoginPage() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const login = useAuthStore((state) => state.login);
  const isAuthenticated = useIsAuthenticated();
  const navigate = useNavigate();

  // If already authenticated, redirect to dashboard
  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard', { replace: true });
    }
  }, [isAuthenticated, navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    if (!username || !password) {
      setError('Please enter username and password.');
      setLoading(false);
      return;
    }

    try {
      const response = await authService.login({ username, password });
      console.log('Login successful:', response.data);

      const { token, user } = response.data; // Assuming backend returns token and user object

      if (token && user) {
        // Update Zustand store
        login(token, user);
        // Navigate to dashboard (will happen automatically via useEffect, but can be explicit too)
        // navigate('/dashboard'); 
      } else {
        setError('Login failed: Invalid response from server.');
      }
    } catch (err: any) {
      console.error('Login error:', err);
      if (err.response && err.response.data && err.response.data.error) {
        setError(`Login failed: ${err.response.data.error}`);
      } else {
        setError('Login failed: An unexpected error occurred.');
      }
    } finally {
      setLoading(false);
    }
  };

  // Alternative redirection check at render time
  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }

  return (
    <div>
      <h2>Login</h2>
      <form onSubmit={handleSubmit}>
        <div>
          <label htmlFor="username">Username:</label>
          <input
            type="text"
            id="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
            disabled={loading}
          />
        </div>
        <div>
          <label htmlFor="password">Password:</label>
          <input
            type="password"
            id="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            disabled={loading}
          />
        </div>
        {error && <p style={{ color: 'red' }}>{error}</p>}
        <button type="submit" disabled={loading}>
          {loading ? 'Logging in...' : 'Login'}
        </button>
      </form>
      {/* TODO: Add link to Signup page */}
    </div>
  );
}

export default LoginPage; 