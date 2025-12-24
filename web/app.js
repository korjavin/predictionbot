// Initialize Telegram Web App
if (typeof window.Telegram !== 'undefined' && window.Telegram.WebApp) {
    window.Telegram.WebApp.ready();
    window.Telegram.WebApp.expand();
}

// Function to fetch user profile with auth
async function fetchUserProfile() {
    const response = await fetch('/api/me', {
        headers: { 'X-Telegram-Init-Data': window.Telegram.WebApp.initData }
    });
    if (!response.ok) throw new Error('Failed to fetch user');
    return response.json();
}

// Format balance as currency
function formatBalance(balance) {
    return balance.toFixed(2);
}

// Display user profile data
async function displayUserProfile() {
    const userNameEl = document.getElementById('user-name');
    const userBalanceEl = document.getElementById('user-balance');
    const loadingEl = document.querySelector('.loading');
    const profileEl = document.getElementById('user-profile');
    
    try {
        const user = await fetchUserProfile();
        loadingEl.style.display = 'none';
        profileEl.style.display = 'block';
        userNameEl.textContent = user.first_name;
        userBalanceEl.textContent = formatBalance(user.balance);
    } catch (error) {
        console.error('Failed to load user profile:', error);
        loadingEl.style.display = 'none';
        userNameEl.textContent = 'Guest';
        userBalanceEl.textContent = '0.00';
    }
}

// Run on page load
document.addEventListener('DOMContentLoaded', displayUserProfile);
