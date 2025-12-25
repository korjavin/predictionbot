# API Test Suite Plan

## Overview
Comprehensive test coverage for all 12 API endpoints to ensure contract safety before refactoring.

## API Endpoints Summary

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/ping` | GET | No | Health check |
| `/api/me` | GET | Yes | Get user profile |
| `/api/me/bailout` | POST | Yes | Request bailout (balance < 1) |
| `/api/me/bets` | GET | Yes | Get user's betting history |
| `/api/me/stats` | GET | Yes | Get user statistics |
| `/api/leaderboard` | GET | No | Get top 20 users |
| `/api/markets` | GET | No | List active markets |
| `/api/markets` | POST | Yes | Create new market |
| `/api/markets/{id}/resolve` | POST | Yes* | Resolve market (creator) |
| `/api/markets/{id}/dispute` | POST | Yes | Raise dispute |
| `/api/admin/resolve` | POST | Yes+ | Admin force resolve |
| `/api/bets` | POST | Yes | Place a bet |

*Creator only, +Admin only

## Test Coverage Matrix

### 1. PING (`/api/ping`)
| Test Case | Expected Status | Description |
|-----------|-----------------|-------------|
| `TestPingHandler` | 200 | Basic health check |

### 2. USER PROFILE (`/api/me`)
| Test Case | Expected Status | Description |
|-----------|-----------------|-------------|
| `TestHandleMeUnauthorized` | 401 | No auth context |
| `TestHandleMeAuthorized` | 200 | Valid auth context |
| `TestHandleMeUserNotFound` | 404 | User deleted from DB |

### 3. BAILOUT (`/api/me/bailout`)
| Test Case | Expected Status | Description |
|-----------|-----------------|-------------|
| `TestHandleBailoutUnauthorized` | 401 | No auth context |
| `TestHandleBailoutBalanceTooHigh` | 400 | Balance >= 1 |
| `TestHandleBailoutCooldownActive` | 429 | Within 24h cooldown |
| `TestHandleBailoutSuccess` | 200 | Eligible user, balance reset |

### 4. USER BETS (`/api/me/bets`)
| Test Case | Expected Status | Description |
|-----------|-----------------|-------------|
| `TestHandleUserBetsUnauthorized` | 401 | No auth context |
| `TestHandleUserBetsEmpty` | 200 | No bets placed |
| `TestHandleUserBetsWithData` | 200 | Returns bet history |

### 5. USER STATS (`/api/me/stats`)
| Test Case | Expected Status | Description |
|-----------|-----------------|-------------|
| `TestHandleUserStatsUnauthorized` | 401 | No auth context |
| `TestHandleUserStatsEmpty` | 200 | New user stats |
| `TestHandleUserStatsWithData` | 200 | User with bets |

### 6. LEADERBOARD (`/api/leaderboard`)
| Test Case | Expected Status | Description |
|-----------|-----------------|-------------|
| `TestHandleLeaderboardEmpty` | 200 | No users |
| `TestHandleLeaderboardWithData` | 200 | Returns sorted users |

### 7. LIST MARKETS (`/api/markets` - GET)
| Test Case | Expected Status | Description |
|-----------|-----------------|-------------|
| `TestHandleListMarketsEmpty` | 200 | No markets |
| `TestHandleListMarketsWithData` | 200 | Returns markets with pools |
| `TestHandleListMarketsCreatorName` | 200 | Creator name displays correctly |

### 8. CREATE MARKET (`/api/markets` - POST)
| Test Case | Expected Status | Description |
|-----------|-----------------|-------------|
| `TestHandleCreateMarketUnauthorized` | 401 | No auth context |
| `TestHandleCreateMarketInvalidBody` | 400 | Malformed JSON |
| `TestHandleCreateMarketShortQuestion` | 400 | < 10 chars |
| `TestHandleCreateMarketLongQuestion` | 400 | > 140 chars |
| `TestHandleCreateMarketInvalidExpiry` | 400 | Invalid RFC3339 |
| `TestHandleCreateMarketPastExpiry` | 400 | Expired date |
| `TestHandleCreateMarketTooSoon` | 400 | < 1 hour from now |
| `TestHandleCreateMarketSuccess` | 201 | Valid request |

### 9. RESOLVE MARKET (`/api/markets/{id}/resolve`)
| Test Case | Expected Status | Description |
|-----------|-----------------|-------------|
| `TestHandleResolveUnauthorized` | 401 | No auth context |
| `TestHandleResolveInvalidMethod` | 405 | Wrong HTTP method |
| `TestHandleResolveInvalidBody` | 400 | Missing outcome |
| `TestHandleResolveInvalidOutcome` | 400 | Not YES/NO |
| `TestHandleResolveNotFound` | 404 | Market doesn't exist |
| `TestHandleResolveNotCreator` | 403 | User not market creator |
| `TestHandleResolveAlreadyResolved` | 409 | Market finalized |
| `TestHandleResolveSuccess` | 200 | Creator resolves their market |

### 10. DISPUTE MARKET (`/api/markets/{id}/dispute`)
| Test Case | Expected Status | Description |
|-----------|-----------------|-------------|
| `TestHandleDisputeUnauthorized` | 401 | No auth context |
| `TestHandleDisputeInvalidMethod` | 405 | Wrong HTTP method |
| `TestHandleDisputeNotFound` | 404 | Market doesn't exist |
| `TestHandleDisputeAlreadyResolved` | 409 | Cannot dispute resolved |
| `TestHandleDisputeSuccess` | 200 | Dispute raised |

### 11. ADMIN RESOLVE (`/api/admin/resolve`)
| Test Case | Expected Status | Description |
|-----------|-----------------|-------------|
| `TestHandleAdminResolveUnauthorized` | 401 | No auth context |
| `TestHandleAdminResolveNotAdmin` | 403 | Non-admin user |
| `TestHandleAdminResolveInvalidBody` | 400 | Invalid request |
| `TestHandleAdminResolveNotFound` | 404 | Market doesn't exist |
| `TestHandleAdminResolveSuccess` | 200 | Admin force resolves |

### 12. PLACE BET (`/api/bets`)
| Test Case | Expected Status | Description |
|-----------|-----------------|-------------|
| `TestHandleBetsUnauthorized` | 401 | No auth context |
| `TestHandleBetsInvalidBody` | 400 | Malformed JSON |
| `TestHandleBetsInvalidOutcome` | 400 | Not YES/NO |
| `TestHandleBetsInvalidAmount` | 400 | Amount <= 0 |
| `TestHandleBetsMarketNotFound` | 404 | Market doesn't exist |
| `TestHandleBetsMarketNotActive` | 403 | Market expired/locked |
| `TestHandleBetsInsufficientFunds` | 402 | Balance < amount |
| `TestHandleBetsSuccess` | 201 | Valid bet placed |

## Response Schema Validation

For each endpoint, tests should verify:
- Status code
- Content-Type header
- JSON response structure (field types)
- Required fields present

Example for `/api/me`:
```go
{
  "id": int64,
  "telegram_id": int64,
  "username": string,
  "first_name": string,
  "balance": int64,
  "balance_display": string
}
```

## Test Utilities to Create

1. `createTestUser()` - Helper to create user with specific balance
2. `createTestMarket()` - Helper to create market with specific expiry
3. `createTestBet()` - Helper to place a bet
4. `assertResponseSchema()` - Helper to validate JSON structure

## Implementation Priority

1. **Priority 1** - Critical path tests (most used endpoints):
   - List Markets, Create Market, Place Bet, Get Me

2. **Priority 2** - Important but less frequent:
   - User Bets, User Stats, Bailout, Resolve

3. **Priority 3** - Admin and edge cases:
   - Admin Resolve, Dispute, Leaderboard

## Success Criteria

- All 12 endpoints have at least one happy-path test
- All auth-protected endpoints have unauthorized test
- All validation logic has boundary tests
- Response schemas are validated
- Tests run in < 5 seconds total
