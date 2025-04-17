import React, { useState, useEffect } from 'react';
import { useNavigate, Navigate } from 'react-router-dom';
import { useAuthStore, useIsAuthenticated } from '../store/authStore';
import { authService } from '../services/api';

function SignupPage() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
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

    if (password !== confirmPassword) {
      setError('Passwords do not match.');
      return;
    }

    if (!username || !password) {
        setError('Please enter username and password.');
        return;
    }

    setLoading(true);

    try {
      const response = await authService.signup({ username, password });
      console.log('Signup successful:', response.data);

      const { token, user } = response.data; // Assuming backend returns token and user upon signup

      if (token && user) {
        // Login user immediately after successful signup
        login(token, user);
        // Navigate to dashboard (will happen automatically via useEffect)
        // navigate('/dashboard');
      } else {
        setError('Signup successful, but failed to log in automatically. Please try logging in.');
      }
    } catch (err: any) {
      console.error('Signup error:', err);
      if (err.response && err.response.data && err.response.data.error) {
        setError(`Signup failed: ${err.response.data.error}`);
      } else {
        setError('Signup failed: An unexpected error occurred.');
      }
    } finally {
      setLoading(false);
    }
  };

  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }

  return (
    <div>
      <h2>Sign Up</h2>
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
        <div>
          <label htmlFor="confirmPassword">Confirm Password:</label>
          <input
            type="password"
            id="confirmPassword"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            required
            disabled={loading}
          />
        </div>
        {error && <p style={{ color: 'red' }}>{error}</p>}
        <button type="submit" disabled={loading}>
          {loading ? 'Signing up...' : 'Sign Up'}
        </button>
      </form>
      {/* TODO: Add link to Login page */}
    </div>
  );
}

export default SignupPage; 