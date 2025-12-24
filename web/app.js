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

// Format date for display
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
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
        
        // Load markets after user profile is loaded
        renderMarkets();
    } catch (error) {
        console.error('Failed to load user profile:', error);
        loadingEl.style.display = 'none';
        userNameEl.textContent = 'Guest';
        userBalanceEl.textContent = '0.00';
    }
}

// Fetch markets from API
async function fetchMarkets() {
    const response = await fetch('/api/markets', {
        headers: { 'X-Telegram-Init-Data': window.Telegram.WebApp.initData }
    });
    if (!response.ok) throw new Error('Failed to fetch markets');
    return response.json();
}

// Render markets to the DOM
async function renderMarkets() {
    const marketsListEl = document.getElementById('markets-list');
    const marketFeedEl = document.getElementById('market-feed');
    
    try {
        const markets = await fetchMarkets();
        marketFeedEl.classList.add('active');
        
        if (markets.length === 0) {
            marketsListEl.innerHTML = '<div class="no-markets">No active markets yet. Create one!</div>';
            return;
        }
        
        marketsListEl.innerHTML = markets.map(market => `
            <div class="market-card">
                <div class="market-question">${escapeHtml(market.question)}</div>
                <div class="market-meta">
                    <span class="market-creator">By ${escapeHtml(market.creator_name)}</span>
                    <span class="market-deadline">${formatDate(market.expires_at)}</span>
                </div>
            </div>
        `).join('');
    } catch (error) {
        console.error('Failed to render markets:', error);
        marketsListEl.innerHTML = '<div class="error-message">Failed to load markets</div>';
    }
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Create a new market
async function createMarket(question, expiresAt) {
    const response = await fetch('/api/markets', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-Telegram-Init-Data': window.Telegram.WebApp.initData
        },
        body: JSON.stringify({
            question: question,
            expires_at: expiresAt
        })
    });
    
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.message || 'Failed to create market');
    }
    
    return response.json();
}

// Set up form event listeners
function setupMarketForm() {
    const createBtn = document.getElementById('create-market-btn');
    const form = document.getElementById('create-market-form');
    const questionInput = document.getElementById('market-question');
    const deadlineInput = document.getElementById('market-deadline');
    const submitBtn = document.getElementById('submit-market-btn');
    const cancelBtn = document.getElementById('cancel-market-btn');
    const messageEl = document.getElementById('form-message');
    
    // Set minimum deadline to 1 hour from now
    const minDate = new Date(Date.now() + 60 * 60 * 1000);
    deadlineInput.min = minDate.toISOString().slice(0, 16);
    
    // Show form
    createBtn.addEventListener('click', () => {
        form.classList.add('active');
        createBtn.style.display = 'none';
        questionInput.focus();
    });
    
    // Hide form
    cancelBtn.addEventListener('click', () => {
        form.classList.remove('active');
        createBtn.style.display = 'inline-block';
        clearForm();
    });
    
    // Submit form
    submitBtn.addEventListener('click', async () => {
        const question = questionInput.value.trim();
        const deadline = deadlineInput.value;
        
        // Validation
        if (question.length < 10 || question.length > 140) {
            messageEl.innerHTML = '<div class="error-message">Question must be between 10 and 140 characters</div>';
            return;
        }
        
        if (!deadline) {
            messageEl.innerHTML = '<div class="error-message">Please select a deadline</div>';
            return;
        }
        
        // Convert to RFC3339 format
        const expiresAt = new Date(deadline).toISOString();
        
        try {
            submitBtn.disabled = true;
            submitBtn.textContent = 'Creating...';
            
            await createMarket(question, expiresAt);
            
            messageEl.innerHTML = '<div class="success-message">Market created successfully!</div>';
            
            // Clear form and refresh markets
            setTimeout(() => {
                form.classList.remove('active');
                createBtn.style.display = 'inline-block';
                clearForm();
                renderMarkets();
            }, 1000);
        } catch (error) {
            messageEl.innerHTML = `<div class="error-message">${escapeHtml(error.message)}</div>`;
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Create';
        }
    });
}

function clearForm() {
    document.getElementById('market-question').value = '';
    document.getElementById('market-deadline').value = '';
    document.getElementById('form-message').innerHTML = '';
}

// Run on page load
document.addEventListener('DOMContentLoaded', () => {
    displayUserProfile();
    setupMarketForm();
});
