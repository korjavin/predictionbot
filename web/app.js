// Initialize Telegram Web App
let telegramWebApp = null;
let initData = '';

if (typeof window.Telegram !== 'undefined' && window.Telegram.WebApp) {
    telegramWebApp = window.Telegram.WebApp;
    telegramWebApp.ready();
    telegramWebApp.expand();
    initData = telegramWebApp.initData || '';
}

// Current active tab
let currentTab = 'markets';
// Current user for leaderboard comparison
let currentUser = null;

// Show global error message
function showGlobalError(message, details = '') {
    const loadingEl = document.querySelector('.loading');
    if (loadingEl) {
        loadingEl.style.color = '#ff6b6b';
        loadingEl.innerHTML = `
            <div style="text-align: center; padding: 20px;">
                <div style="font-size: 24px; margin-bottom: 12px;">❌</div>
                <div style="font-size: 16px; font-weight: 600; margin-bottom: 8px;">${escapeHtml(message)}</div>
                ${details ? `<div style="font-size: 12px; color: #888; margin-top: 8px;">${escapeHtml(details)}</div>` : ''}
            </div>
        `;
    }
}

// Function to fetch user profile with auth
async function fetchUserProfile() {
    const response = await fetch('/api/me', {
        headers: { 'X-Telegram-Init-Data': initData }
    });
    if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
        throw new Error(errorData.error || `HTTP ${response.status}`);
    }
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

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Display user profile data
async function displayUserProfile() {
    const userNameEl = document.getElementById('user-name');
    const userBalanceEl = document.getElementById('user-balance');
    const loadingEl = document.querySelector('.loading');
    const mainContentEl = document.getElementById('main-content');
    const profileNameEl = document.getElementById('profile-name');
    const profileUsernameEl = document.getElementById('profile-username');
    const profileBalanceEl = document.getElementById('profile-balance');
    const profileAvatarEl = document.getElementById('profile-avatar');

    // Check if Telegram WebApp is available
    if (!telegramWebApp) {
        showGlobalError(
            'Telegram WebApp Not Available',
            'This app must be opened inside Telegram. Please use a Telegram bot to access this app.'
        );
        return;
    }

    // Check if initData is present
    if (!initData || initData.trim() === '') {
        showGlobalError(
            'Authentication Failed',
            'No Telegram authentication data found. Please restart the app from your Telegram bot.'
        );
        return;
    }

    try {
        const user = await fetchUserProfile();
        loadingEl.style.display = 'none';
        mainContentEl.style.display = 'block';

        // Store current user for leaderboard comparison
        currentUser = user;

        // Update header name (if it exists)
        if (userNameEl) {
            userNameEl.textContent = user.first_name;
        }

        // Update balance in header
        userBalanceEl.textContent = formatBalance(user.balance) + ' WSC';

        // Update profile tab
        profileNameEl.textContent = user.first_name;
        profileBalanceEl.textContent = formatBalance(user.balance) + ' WSC';

        // Set avatar initial
        const initial = user.first_name ? user.first_name.charAt(0).toUpperCase() : '?';
        profileAvatarEl.textContent = initial;

        if (user.username) {
            profileUsernameEl.textContent = '@' + user.username;
        } else {
            profileUsernameEl.textContent = '';
        }

        // Render mortgage button if balance is low
        renderMortgageButton();

        // Load initial tab content
        if (currentTab === 'markets') {
            renderMarkets();
        } else if (currentTab === 'leaders') {
            renderLeaderboard();
        } else if (currentTab === 'profile') {
            renderProfile();
        }

        // Set up navigation tabs
        setupNavigation();
    } catch (error) {
        console.error('Failed to load user profile:', error);
        showGlobalError(
            'Failed to Load Profile',
            error.message || 'Unable to connect to server. Please check your connection and try again.'
        );
    }
}

// Set up navigation tabs
function setupNavigation() {
    const tabs = document.querySelectorAll('.nav-tab');
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const tabName = tab.dataset.tab;
            
            // Update active tab
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            
            // Show/hide content
            document.getElementById('markets-tab').style.display = tabName === 'markets' ? 'block' : 'none';
            document.getElementById('leaders-tab').style.display = tabName === 'leaders' ? 'block' : 'none';
            document.getElementById('profile-tab').style.display = tabName === 'profile' ? 'block' : 'none';
            
            currentTab = tabName;
            
            // Load content for the tab
            if (tabName === 'markets') {
                renderMarkets();
            } else if (tabName === 'leaders') {
                renderLeaderboard();
            } else if (tabName === 'profile') {
                renderProfile();
            }
        });
    });
}

// Render profile tab (stats and history)
async function renderProfile() {
    await Promise.all([
        renderUserStats(),
        renderBetHistory()
    ]);
}

// Fetch and display user stats
async function renderUserStats() {
    try {
        const response = await fetch('/api/me/stats', {
            headers: { 'X-Telegram-Init-Data': initData }
        });
        
        if (!response.ok) throw new Error('Failed to fetch stats');
        
        const stats = await response.json();
        
        document.getElementById('stat-total-bets').textContent = stats.total_bets || 0;
        document.getElementById('stat-wins').textContent = stats.wins || 0;
        document.getElementById('stat-win-rate').textContent = (stats.win_rate || 0).toFixed(1) + '%';
        document.getElementById('stat-profit').textContent = formatBalance(stats.total_wins - stats.total_wager);
        
    } catch (error) {
        console.error('Failed to render stats:', error);
        document.getElementById('stat-total-bets').textContent = '-';
        document.getElementById('stat-wins').textContent = '-';
        document.getElementById('stat-win-rate').textContent = '-';
        document.getElementById('stat-profit').textContent = '-';
    }
}

// Fetch and display bet history
async function renderBetHistory() {
    const historyListEl = document.getElementById('history-list');

    try {
        const response = await fetch('/api/me/bets', {
            headers: { 'X-Telegram-Init-Data': initData }
        });
        
        if (!response.ok) throw new Error('Failed to fetch bet history');
        
        const bets = await response.json();
        
        if (bets.length === 0) {
            historyListEl.innerHTML = '<div class="no-markets">No bets placed yet. Start predicting!</div>';
            return;
        }
        
        historyListEl.innerHTML = bets.map(bet => {
            const statusClass = 'status-' + bet.status.toLowerCase();
            const amountWSC = formatBalance(bet.amount);
            const payoutWSC = bet.payout ? formatBalance(bet.payout) : null;
            
            let resultText = '';
            if (bet.status === 'WON') {
                resultText = `<span class="history-payout" style="color: #4ade80;">+${payoutWSC} WSC</span>`;
            } else if (bet.status === 'REFUNDED') {
                resultText = `<span class="history-payout" style="color: #aaaaaa;">Refunded</span>`;
            } else if (bet.status === 'LOST') {
                resultText = `<span class="history-payout" style="color: #ff6b6b;">-${amountWSC} WSC</span>`;
            } else {
                resultText = `<span class="history-payout" style="color: #aaaacc;">${amountWSC} WSC</span>`;
            }
            
            return `
                <div class="history-card">
                    <div class="history-info">
                        <div class="history-question">${escapeHtml(bet.question)}</div>
                        <div class="history-meta">
                            Bet ${bet.outcome_chosen} • ${formatDate(bet.placed_at)}
                        </div>
                        <span class="status-badge ${statusClass}">${bet.status}</span>
                    </div>
                    <div class="history-amount">
                        ${resultText}
                    </div>
                </div>
            `;
        }).join('');
        
    } catch (error) {
        console.error('Failed to render bet history:', error);
        historyListEl.innerHTML = '<div class="error-message">Failed to load bet history</div>';
    }
}

// Fetch markets from API
async function fetchMarkets() {
    const response = await fetch('/api/markets', {
        headers: { 'X-Telegram-Init-Data': initData }
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
        marketFeedEl.classList.add('visible');
        
        if (markets.length === 0) {
            marketsListEl.innerHTML = '<div class="no-markets">No active markets yet. Create one!</div>';
            return;
        }
        
        const now = new Date();
        marketsListEl.innerHTML = markets.map(market => {
            const expiresAt = new Date(market.expires_at);
            const isExpired = now >= expiresAt;
            const totalPool = (market.pool_yes || 0) + (market.pool_no || 0);
            const yesPercent = totalPool > 0 ? ((market.pool_yes || 0) / totalPool * 100).toFixed(0) : 50;
            const noPercent = totalPool > 0 ? ((market.pool_no || 0) / totalPool * 100).toFixed(0) : 50;
            const isCreator = currentUser && market.creator_id === currentUser.id;
            const isLocked = market.status === 'LOCKED';
            const canResolve = isCreator && isLocked;
            
            return `
                <div class="market-card" id="market-${market.id}">
                    <div class="market-question">${escapeHtml(market.question)}</div>
                    <div class="market-meta">
                        <span class="market-creator">By ${escapeHtml(market.creator_name)}</span>
                        <span class="market-deadline">${formatDate(market.expires_at)}</span>
                    </div>
                    <div class="market-odds">
                        <span class="odds-yes">YES ${yesPercent}%</span>
                        <span class="odds-separator">|</span>
                        <span class="odds-no">NO ${noPercent}%</span>
                    </div>
                    ${canResolve ? `
                    <div class="resolve-section" id="resolve-section-${market.id}">
                        <div class="resolve-section-title">🎯 Resolve Market</div>
                        <div class="resolve-question">${escapeHtml(market.question)}</div>
                        <div class="resolve-buttons">
                            <button class="resolve-btn resolve-btn-yes"
                                    data-market="${market.id}"
                                    data-outcome="YES">
                                Resolve YES
                            </button>
                            <button class="resolve-btn resolve-btn-no"
                                    data-market="${market.id}"
                                    data-outcome="NO">
                                Resolve NO
                            </button>
                        </div>
                        <div class="resolve-message" id="resolve-message-${market.id}"></div>
                    </div>
                    ` : ''}
                    <div class="betting-ui ${isExpired || isLocked ? 'disabled' : ''}" id="betting-ui-${market.id}">
                        <div class="bet-amount-group">
                            <input type="number" 
                                   id="bet-amount-${market.id}" 
                                   placeholder="Amount" 
                                   min="1" 
                                   ${isExpired || isLocked ? 'disabled' : ''}>
                        </div>
                        <div class="bet-buttons">
                            <button class="btn btn-yes bet-btn"
                                    data-market="${market.id}"
                                    data-outcome="YES"
                                    ${isExpired || isLocked ? 'disabled' : ''}>
                                YES<br><small>${formatBalance(market.pool_yes || 0)}</small>
                            </button>
                            <button class="btn btn-no bet-btn"
                                    data-market="${market.id}"
                                    data-outcome="NO"
                                    ${isExpired || isLocked ? 'disabled' : ''}>
                                NO<br><small>${formatBalance(market.pool_no || 0)}</small>
                            </button>
                        </div>
                        <div class="bet-message" id="bet-message-${market.id}"></div>
                    </div>
                </div>
            `;
        }).join('');
        
        // Add click handlers for bet buttons
        document.querySelectorAll('.bet-btn').forEach(btn => {
            btn.addEventListener('click', handleBetClick);
        });
        
        // Add click handlers for resolve buttons
        document.querySelectorAll('.resolve-btn').forEach(btn => {
            btn.addEventListener('click', handleResolveClick);
        });
    } catch (error) {
        console.error('Failed to render markets:', error);
        marketsListEl.innerHTML = '<div class="error-message">Failed to load markets</div>';
    }
}

// Handle YES/NO bet button clicks
async function handleBetClick(event) {
    const btn = event.currentTarget;
    const marketId = parseInt(btn.dataset.market, 10);
    const outcome = btn.dataset.outcome;
    const amountInput = document.getElementById(`bet-amount-${marketId}`);
    const messageEl = document.getElementById(`bet-message-${marketId}`);
    const bettingUi = document.getElementById(`betting-ui-${marketId}`);
    
    const amount = parseInt(amountInput.value, 10);
    const balance = parseFloat(document.getElementById('user-balance').textContent);
    
    // Validation
    if (!amount || amount < 1) {
        messageEl.innerHTML = '<div class="error-message">Please enter a valid amount (minimum 1)</div>';
        return;
    }
    
    if (amount > balance) {
        messageEl.innerHTML = '<div class="error-message">Insufficient balance</div>';
        return;
    }
    
    // Show loading state
    btn.disabled = true;
    btn.textContent = 'Placing...';
    messageEl.innerHTML = '';
    
    try {
        const result = await placeBet(marketId, outcome, amount);
        
        // Show success
        messageEl.innerHTML = `<div class="success-message">Bet placed! New balance: ${formatBalance(result.new_balance)}</div>`;
        
        // Update balance display
        document.getElementById('user-balance').textContent = formatBalance(result.new_balance) + ' WSC';
        document.getElementById('profile-balance').textContent = formatBalance(result.new_balance) + ' WSC';
        
        // Refresh markets to show updated pools
        await renderMarkets();
        
    } catch (error) {
        messageEl.innerHTML = `<div class="error-message">${escapeHtml(error.message)}</div>`;

        // Restore button state
        btn.disabled = false;
        btn.innerHTML = `${outcome}<br><small>Pool: loading...</small>`;

        // Refresh markets to restore correct pool amounts
        await renderMarkets();
    }
}

// Place a bet on a market
async function placeBet(marketId, outcome, amount) {
    const response = await fetch('/api/bets', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-Telegram-Init-Data': initData
        },
        body: JSON.stringify({
            market_id: marketId,
            outcome: outcome,
            amount: amount
        })
    });

    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to place bet');
    }

    return response.json();
}

// Resolve a market (owner only, when LOCKED)
async function resolveMarket(marketId, outcome) {
    const response = await fetch(`/api/markets/${marketId}/resolve`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-Telegram-Init-Data': initData
        },
        body: JSON.stringify({
            outcome: outcome
        })
    });

    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.message || error.error || 'Failed to resolve market');
    }

    return response.json();
}

// Handle YES/NO resolve button clicks
async function handleResolveClick(event) {
    const btn = event.currentTarget;
    const marketId = parseInt(btn.dataset.market, 10);
    const outcome = btn.dataset.outcome;
    const messageEl = document.getElementById(`resolve-message-${marketId}`);
    const resolveSection = document.getElementById(`resolve-section-${marketId}`);
    
    // Disable button and show loading state
    btn.disabled = true;
    const originalText = btn.textContent;
    btn.textContent = 'Resolving...';
    messageEl.innerHTML = '';
    
    try {
        await resolveMarket(marketId, outcome);
        
        // Show success message
        messageEl.innerHTML = `<div class="success-message">✓ Market resolved to ${outcome}!</div>`;
        
        // Provide haptic feedback
        if (telegramWebApp) {
            telegramWebApp.HapticFeedback.notificationOccurred('success');
        }
        
        // Refresh markets to show updated status
        await renderMarkets();
        
    } catch (error) {
        // Show error message
        messageEl.innerHTML = `<div class="error-message">${escapeHtml(error.message)}</div>`;
        
        // Restore button state
        btn.disabled = false;
        btn.textContent = originalText;
        
        // Provide haptic feedback for error
        if (telegramWebApp) {
            telegramWebApp.HapticFeedback.notificationOccurred('error');
        }
    }
}

// Create a new market
async function createMarket(question, expiresAt) {
    const response = await fetch('/api/markets', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-Telegram-Init-Data': initData
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

// Render mortgage button based on user balance
function renderMortgageButton() {
    const mortgageBtn = document.getElementById('mortgage-btn');
    const mortgageInfo = document.getElementById('mortgage-info');
    const mortgageMessage = document.getElementById('mortgage-message');
    
    if (!mortgageBtn || !currentUser) return;
    
    // Show button if balance < 1
    if (currentUser.balance < 1) {
        mortgageBtn.style.display = 'block';
        mortgageInfo.style.display = 'block';
    } else {
        mortgageBtn.style.display = 'none';
        mortgageInfo.style.display = 'none';
    }
    
    // Set up mortgage button click handler
    mortgageBtn.onclick = handleMortgageClick;
}

// Handle mortgage button click
async function handleMortgageClick() {
    const mortgageBtn = document.getElementById('mortgage-btn');
    const mortgageMessage = document.getElementById('mortgage-message');
    
    mortgageBtn.disabled = true;
    mortgageBtn.textContent = 'Processing...';
    mortgageMessage.innerHTML = '';
    
    try {
        const result = await takeMortgage();
        
        // Show success message
        mortgageMessage.innerHTML = `<div class="success-message">${escapeHtml(result.message)}! New balance: ${formatBalance(result.new_balance)} WSC</div>`;
        
        // Play success sound (optional)
        if (telegramWebApp) {
            telegramWebApp.HapticFeedback.notificationOccurred('success');
        }
        
        // Refresh user profile to update balance
        const user = await fetchUserProfile();
        currentUser = user;
        
        // Update balance displays
        document.getElementById('user-balance').textContent = formatBalance(user.balance) + ' WSC';
        document.getElementById('profile-balance').textContent = formatBalance(user.balance) + ' WSC';
        
        // Hide mortgage button after successful bailout
        renderMortgageButton();
        
    } catch (error) {
        // Show error message
        mortgageMessage.innerHTML = `<div class="error-message">${escapeHtml(error.message)}</div>`;
        
        // Restore button state
        mortgageBtn.disabled = false;
        mortgageBtn.textContent = '💸 Take Mortgage';
        
        // Haptic feedback for error
        if (telegramWebApp) {
            telegramWebApp.HapticFeedback.notificationOccurred('error');
        }
    }
}

// Take mortgage (call bailout API)
async function takeMortgage() {
    const response = await fetch('/api/me/bailout', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-Telegram-Init-Data': initData
        }
    });
    
    if (!response.ok) {
        const error = await response.json();
        
        if (error.error === 'cooldown_active') {
            throw new Error('Bank says NO: ' + (error.next_available || 'Come back later'));
        }
        if (error.error === 'balance_too_high') {
            throw new Error('You have sufficient funds - mortgage not needed!');
        }
        throw new Error(error.error || 'Failed to take mortgage');
    }
    
    return response.json();
}

// Fetch leaderboard from API
async function fetchLeaderboard() {
    const response = await fetch('/api/leaderboard', {
        headers: { 'X-Telegram-Init-Data': initData }
    });
    if (!response.ok) throw new Error('Failed to fetch leaderboard');
    return response.json();
}

// Render leaderboard to the DOM
async function renderLeaderboard() {
    const leaderboardListEl = document.getElementById('leaderboard-list');
    const leaderboardFeedEl = document.getElementById('leaderboard-feed');
    
    try {
        const leaderboard = await fetchLeaderboard();
        leaderboardFeedEl.classList.add('visible');
        
        if (leaderboard.length === 0) {
            leaderboardListEl.innerHTML = '<div class="no-markets">No leaders yet. Be the first!</div>';
            return;
        }
        
        leaderboardListEl.innerHTML = leaderboard.map(entry => {
            const isMe = currentUser && entry.name === currentUser.first_name;
            
            // Get medal or rank
            let rankDisplay = '';
            let rankClass = '';
            let badge = '';
            
            if (entry.rank === 1) {
                rankDisplay = '🥇';
                rankClass = 'gold';
                badge = '<span class="leaderboard-badge">🥇</span>';
            } else if (entry.rank === 2) {
                rankDisplay = '2';
                rankClass = 'silver';
                badge = '<span class="leaderboard-badge">🥈</span>';
            } else if (entry.rank === 3) {
                rankDisplay = '3';
                rankClass = 'bronze';
                badge = '<span class="leaderboard-badge">🥉</span>';
            } else {
                rankDisplay = entry.rank;
            }
            
            const name = escapeHtml(entry.name);
            const username = entry.username ? '@' + escapeHtml(entry.username) : '';
            
            return `
                <div class="leaderboard-card ${isMe ? 'is-me' : ''}">
                    <div class="leaderboard-rank ${rankClass}">${rankDisplay}</div>
                    ${badge}
                    <div class="leaderboard-info">
                        <div class="leaderboard-name">${name}${isMe ? ' (You)' : ''}</div>
                        <div class="leaderboard-username">${username}</div>
                    </div>
                    <div class="leaderboard-balance">${entry.balance_display} WSC</div>
                </div>
            `;
        }).join('');
        
    } catch (error) {
        console.error('Failed to render leaderboard:', error);
        leaderboardListEl.innerHTML = '<div class="error-message">Failed to load leaderboard</div>';
    }
}

// Run on page load
document.addEventListener('DOMContentLoaded', () => {
    displayUserProfile();
    setupMarketForm();
});
