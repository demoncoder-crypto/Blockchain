import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

// Define the shape of the user object (adjust based on backend response)
interface User {
  id: string; // Assuming UUID comes as string
  username: string;
  // Add other relevant user fields if needed
}

interface AuthState {
  token: string | null;
  user: User | null;
  isAuthenticated: boolean;
  login: (token: string, user: User) => void;
  logout: () => void;
  setToken: (token: string | null) => void; // Allow setting token directly (e.g., on initial load)
}

export const useAuthStore = create<AuthState>()(
  // Persist state to localStorage
  persist(
    (set) => ({
      token: null,
      user: null,
      isAuthenticated: false,

      login: (token, user) => {
        set({ token, user, isAuthenticated: true });
        // Optional: Update axios default header immediately if needed,
        // although interceptor handles subsequent requests.
        // apiClient.defaults.headers.common['Authorization'] = `Bearer ${token}`;
        console.log('Logged in:', user);
      },

      logout: () => {
        set({ token: null, user: null, isAuthenticated: false });
        // delete apiClient.defaults.headers.common['Authorization'];
        console.log('Logged out');
        // Consider redirecting to login page here or in a component effect
        // window.location.href = '/login'; 
      },

      setToken: (token) => {
        // Simple token setter, doesn't imply full login (e.g., no user info)
        // Useful if token is loaded but user info needs separate fetch.
        set({ token, isAuthenticated: !!token });
      },
    }),
    {
      name: 'auth-storage', // name of the item in storage (must be unique)
      storage: createJSONStorage(() => localStorage), // use localStorage
      // Only persist the token, user can be re-fetched or derived if needed
      partialize: (state) => ({ token: state.token }),
      // onRehydrateStorage: () => (state) => {
      //   // Optional: Perform actions after state is rehydrated
      //   if (state?.token) {
      //     state.setToken(state.token); // Ensure isAuthenticated is set
      //     // TODO: Maybe trigger a fetch for user profile?
      //   }
      // },
    }
  )
);

// --- Selector Hooks (optional but recommended) ---
export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated);
export const useAuthToken = () => useAuthStore((state) => state.token);
export const useAuthUser = () => useAuthStore((state) => state.user);
export const useAuthActions = () => useAuthStore((state) => ({ login: state.login, logout: state.logout })); 