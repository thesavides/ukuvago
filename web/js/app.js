// UkuvaGo Angel Investment Platform - Frontend Application

const API_BASE = '/api';
let currentUser = null;
let authToken = localStorage.getItem('token');

// API Client
const api = {
    async request(method, endpoint, data = null) {
        const headers = { 'Content-Type': 'application/json' };
        if (authToken) headers['Authorization'] = `Bearer ${authToken}`;

        const config = { method, headers };
        if (data) config.body = JSON.stringify(data);

        const res = await fetch(API_BASE + endpoint, config);
        const json = await res.json();

        if (!res.ok) throw new Error(json.error || 'Request failed');
        return json;
    },

    get: (endpoint) => api.request('GET', endpoint),
    post: (endpoint, data) => api.request('POST', endpoint, data),
    put: (endpoint, data) => api.request('PUT', endpoint, data),
    delete: (endpoint) => api.request('DELETE', endpoint)
};

// Auth functions
async function login(email, password) {
    const data = await api.post('/auth/login', { email, password });
    authToken = data.token;
    currentUser = data.user;
    localStorage.setItem('token', authToken);
    return data;
}

async function register(userData) {
    const data = await api.post('/auth/register', userData);
    authToken = data.token;
    currentUser = data.user;
    localStorage.setItem('token', authToken);
    return data;
}

function logout() {
    authToken = null;
    currentUser = null;
    localStorage.removeItem('token');
    showPage('home');
    updateNav();
}

async function checkAuth() {
    if (!authToken) return null;
    try {
        const data = await api.get('/auth/me');
        currentUser = data.user;
        return currentUser;
    } catch {
        logout();
        return null;
    }
}

// Toast notifications
function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container') || createToastContainer();
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    container.appendChild(toast);
    setTimeout(() => toast.remove(), 4000);
}

function createToastContainer() {
    const container = document.createElement('div');
    container.id = 'toast-container';
    container.className = 'toast-container';
    document.body.appendChild(container);
    return container;
}

// Navigation
function updateNav() {
    const authNav = document.getElementById('auth-nav');
    const userNav = document.getElementById('user-nav');

    if (currentUser) {
        authNav?.classList.add('hidden');
        userNav?.classList.remove('hidden');
        document.getElementById('user-name')?.textContent = currentUser.first_name;
        document.getElementById('user-role')?.textContent = currentUser.role;
    } else {
        authNav?.classList.remove('hidden');
        userNav?.classList.add('hidden');
    }
}

// Page routing
const pages = {};

function showPage(pageName, params = {}) {
    document.querySelectorAll('[data-page]').forEach(p => p.classList.add('hidden'));
    const page = document.querySelector(`[data-page="${pageName}"]`);
    if (page) {
        page.classList.remove('hidden');
        if (pages[pageName]) pages[pageName](params);
    }
    window.scrollTo(0, 0);
}

// Format currency
function formatCurrency(amount, currency = 'usd') {
    const symbols = { usd: '$', zar: 'R', eur: '€', gbp: '£' };
    return (symbols[currency] || '$') + amount.toLocaleString();
}

// Initialize app
document.addEventListener('DOMContentLoaded', async () => {
    await checkAuth();
    updateNav();

    // Handle hash navigation
    const hash = window.location.hash.slice(1) || 'home';
    showPage(hash);

    window.addEventListener('hashchange', () => {
        const hash = window.location.hash.slice(1) || 'home';
        showPage(hash);
    });
});

// Form handlers
document.getElementById('login-form')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    try {
        await login(e.target.email.value, e.target.password.value);
        showToast('Welcome back!', 'success');
        showPage(currentUser.role === 'admin' ? 'admin' : currentUser.role + '-dashboard');
        updateNav();
    } catch (err) {
        showToast(err.message, 'error');
    }
});

document.getElementById('register-form')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const formData = new FormData(e.target);
    try {
        await register(Object.fromEntries(formData));
        showToast('Registration successful!', 'success');
        showPage(currentUser.role + '-dashboard');
        updateNav();
    } catch (err) {
        showToast(err.message, 'error');
    }
});

// Load projects
async function loadProjects(category = '') {
    const grid = document.getElementById('projects-grid');
    if (!grid) return;

    grid.innerHTML = '<div class="spinner"></div>';

    try {
        const endpoint = category ? `/projects?category=${category}` : '/projects';
        const data = await api.get(endpoint);

        if (!data.projects?.length) {
            grid.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📋</div><div class="empty-state-title">No projects yet</div></div>';
            return;
        }

        grid.innerHTML = data.projects.map(p => `
      <div class="card project-card" onclick="viewProject('${p.id}')">
        <div class="project-card-image">
          ${p.primary_image ? `<img src="/uploads/${p.primary_image}" alt="${p.title}">` : '<span style="font-size:3rem">🚀</span>'}
        </div>
        <span class="project-card-category">${p.category?.name || 'Uncategorized'}</span>
        <h3 class="project-card-title">${p.title}</h3>
        <p class="project-card-tagline">${p.tagline || ''}</p>
        <div class="project-card-investment">
          <span class="investment-label">Min Investment</span>
          <span class="investment-value">${formatCurrency(p.min_investment)}</span>
        </div>
      </div>
    `).join('');
    } catch (err) {
        grid.innerHTML = `<div class="alert alert-error">${err.message}</div>`;
    }
}

async function viewProject(id) {
    if (!currentUser) {
        showToast('Please log in to view project details', 'warning');
        showPage('login');
        return;
    }
    window.location.hash = `project/${id}`;
}

// Load categories
async function loadCategories() {
    const container = document.getElementById('categories');
    if (!container) return;

    try {
        const data = await api.get('/categories');
        container.innerHTML = data.categories.map(c => `
      <span class="category-badge" onclick="filterByCategory('${c.id}')">
        <span class="icon">${c.icon}</span>
        ${c.name}
      </span>
    `).join('');
    } catch (err) {
        console.error('Failed to load categories:', err);
    }
}

function filterByCategory(categoryId) {
    document.querySelectorAll('.category-badge').forEach(b => b.classList.remove('active'));
    event.target.closest('.category-badge')?.classList.add('active');
    loadProjects(categoryId);
}

// Expose functions globally
window.login = login;
window.logout = logout;
window.showPage = showPage;
window.viewProject = viewProject;
window.loadProjects = loadProjects;
window.loadCategories = loadCategories;
window.filterByCategory = filterByCategory;
