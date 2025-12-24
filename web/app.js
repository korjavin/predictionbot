// Initialize Telegram Web App
if (typeof window.Telegram !== 'undefined' && window.Telegram.WebApp) {
    window.Telegram.WebApp.ready();
    window.Telegram.WebApp.expand();
}

// Function to fetch the ping endpoint with auth
async function checkAuth() {
    const statusEl = document.getElementById('status');
    const loadingEl = document.querySelector('.loading');
    
    try {
        // Get initData from Telegram Web App
        const initData = window.Telegram?.WebApp?.initData || '';
        
        // Make request to /api/ping endpoint
        const response = await fetch('/api/ping', {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                'X-Telegram-Init-Data': initData
            }
        });

        if (response.ok) {
            const data = await response.json();
            loadingEl.style.display = 'none';
            statusEl.className = 'status success';
            statusEl.textContent = `✅ Auth successful! Status: ${data.status}`;
        } else {
            throw new Error(`HTTP ${response.status}`);
        }
    } catch (error) {
        console.log('Auth check failed (expected if not in Telegram):', error.message);
        loadingEl.style.display = 'none';
        statusEl.className = 'status error';
        statusEl.textContent = `ℹ️ Running outside Telegram or auth pending`;
    }
}

// Run auth check on page load
document.addEventListener('DOMContentLoaded', checkAuth);
