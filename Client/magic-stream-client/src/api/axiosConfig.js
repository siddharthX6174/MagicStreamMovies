import axios from 'axios';

// Use localhost:8080 as the default API URL if VITE_API_BASE_URL is not set
const apiUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
console.log('API Base URL:', apiUrl);

export default axios.create({
    baseURL: apiUrl,
    headers: {'Content-Type': 'application/json'},
    withCredentials: true,
})