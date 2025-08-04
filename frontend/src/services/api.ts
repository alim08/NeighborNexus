import axios from 'axios';

// Create axios instance
export const api = axios.create({
  baseURL: process.env.REACT_APP_API_URL || 'http://localhost:8080',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor to handle errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Token expired or invalid
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// API functions
export const authAPI = {
  login: (email: string, password: string) =>
    api.post('/api/v1/auth/login', { email, password }),
  
  register: (userData: any) =>
    api.post('/api/v1/auth/register', userData),
  
  refreshToken: (refreshToken: string) =>
    api.post('/api/v1/auth/refresh', { refresh_token: refreshToken }),
  
  getProfile: () =>
    api.get('/api/v1/profile'),
  
  updateProfile: (updates: any) =>
    api.put('/api/v1/profile', updates),
};

export const needsAPI = {
  createNeed: (needData: any) =>
    api.post('/api/v1/needs/', needData),
  
  getNeeds: (params?: any) =>
    api.get('/api/v1/needs/', { params }),
  
  getNeed: (id: string) =>
    api.get(`/api/v1/needs/${id}`),
  
  updateNeed: (id: string, updates: any) =>
    api.put(`/api/v1/needs/${id}`, updates),
  
  deleteNeed: (id: string) =>
    api.delete(`/api/v1/needs/${id}`),
  
  acceptNeed: (id: string) =>
    api.post(`/api/v1/needs/${id}/accept`),
};

export const volunteersAPI = {
  createProfile: (profileData: any) =>
    api.post('/api/v1/volunteers/profile', profileData),
  
  getProfile: () =>
    api.get('/api/v1/volunteers/profile'),
  
  updateProfile: (updates: any) =>
    api.put('/api/v1/volunteers/profile', updates),
  
  getMatches: () =>
    api.get('/api/v1/volunteers/matches'),
};

export const tasksAPI = {
  getTasks: () =>
    api.get('/api/v1/tasks/'),
  
  getTask: (id: string) =>
    api.get(`/api/v1/tasks/${id}`),
  
  updateTaskStatus: (id: string, status: string, data?: any) =>
    api.put(`/api/v1/tasks/${id}/status`, { status, ...data }),
  
  submitFeedback: (id: string, feedback: any) =>
    api.post(`/api/v1/tasks/${id}/feedback`, feedback),
};

// WebSocket connection helper
export const createWebSocket = (token: string) => {
  const ws = new WebSocket(`ws://localhost:8080/api/v1/ws`);
  
  ws.onopen = () => {
    console.log('WebSocket connected');
    // Send authentication
    ws.send(JSON.stringify({
      type: 'auth',
      token: token,
    }));
  };
  
  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      console.log('WebSocket message:', data);
      // Handle different message types
      switch (data.type) {
        case 'new_need':
          // Handle new need notification
          break;
        case 'need_accepted':
          // Handle need accepted notification
          break;
        case 'task_status_update':
          // Handle task status update
          break;
        default:
          console.log('Unknown message type:', data.type);
      }
    } catch (error) {
      console.error('Error parsing WebSocket message:', error);
    }
  };
  
  ws.onerror = (error) => {
    console.error('WebSocket error:', error);
  };
  
  ws.onclose = () => {
    console.log('WebSocket disconnected');
  };
  
  return ws;
}; 