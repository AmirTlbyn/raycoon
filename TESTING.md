# Raycoon Testing Guide

Complete guide for testing all features of Raycoon.

## Prerequisites

- ✅ Raycoon binary built (`make build`)
- ⚠️ Xray-core binary (optional, for connection testing)
- 📝 Valid proxy URIs (for real testing)

## Quick Test Checklist

- [ ] Config Management (add/list/show/delete)
- [ ] Group Management (create/list/delete)
- [ ] Subscription Management (groups with subs)
- [ ] Database Operations (CRUD)
- [ ] Connection Management (requires Xray)
- [ ] Error Handling

---

## 1. Config Management

### Test 1.1: Add VMess Config

```bash
# Add a test VMess config
./bin/raycoon config add 'vmess://eyJ2IjoiMiIsInBzIjoidGVzdC1zZXJ2ZXIiLCJhZGQiOiIxMjcuMC4wLjEiLCJwb3J0IjoiODA4MCIsImlkIjoiYjgzMTUyMmQtYzZiNC00MjZhLWJmMzQtY2ZhZGY4YTdlNDA0IiwiYWlkIjoiMCIsInNjeSI6ImF1dG8iLCJuZXQiOiJ0Y3AiLCJ0eXBlIjoibm9uZSIsImhvc3QiOiIiLCJwYXRoIjoiIiwidGxzIjoiIn0='

# Expected output:
# ✓ Config added successfully!
# ID: 1
# Name: test-server
# Protocol: vmess
# ...
```

### Test 1.2: Add VLESS Config

```bash
# Add a VLESS config
./bin/raycoon config add 'vless://a7b8c9d0-1234-5678-90ab-cdef12345678@example.com:443?type=ws&security=tls&path=/ws#VLESS-Test'

# Expected: Config added with protocol=vless
```

### Test 1.3: Add Trojan Config

```bash
# Add a Trojan config
./bin/raycoon config add 'trojan://password123@example.com:443?security=tls&type=tcp#Trojan-Test'

# Expected: Config added with protocol=trojan
```

### Test 1.4: Add with Custom Options

```bash
# Add config to specific group with tags
./bin/raycoon config add 'vmess://...' \
  --name "My Custom Server" \
  --group work \
  --tags "fast,reliable,us" \
  --notes "Production server in US-East"

# Expected: Config added to 'work' group with custom metadata
```

### Test 1.5: List Configs

```bash
# List all configs
./bin/raycoon config list

# List configs from specific group
./bin/raycoon config list --group global

# List only enabled configs
./bin/raycoon config list --enabled

# Filter by protocol
./bin/raycoon config list --protocol vmess

# Expected: Table showing configs with columns: ID, NAME, PROTOCOL, ADDRESS, GROUP, ENABLED
```

### Test 1.6: Show Config Details

```bash
# Show by ID
./bin/raycoon config show 1

# Show by name
./bin/raycoon config show "test-server"

# Expected: Detailed config information including:
# - Basic info (ID, name, protocol, address, port)
# - Network and transport settings
# - TLS status
# - Group membership
# - Statistics (use count, last used)
# - Original URI
```

### Test 1.7: Delete Config

```bash
# Delete with confirmation
./bin/raycoon config delete 1
# (press 'N' to cancel)

# Delete by name
./bin/raycoon config delete "test-server"
# (press 'y' to confirm)

# Force delete (no confirmation)
./bin/raycoon config delete 2 --force

# Expected: Config deleted from database
```

**✅ Config Management Test Results:**
- [ ] Add VMess config
- [ ] Add VLESS config
- [ ] Add Trojan config
- [ ] Add with custom options
- [ ] List configs (all, filtered)
- [ ] Show config details
- [ ] Delete config

---

## 2. Group Management

### Test 2.1: Create Simple Group

```bash
# Create a basic group
./bin/raycoon group create work --desc "Work servers"

# Expected: Group created with ID and description
```

### Test 2.2: Create Group with Subscription

```bash
# Create group with subscription URL
./bin/raycoon group create personal \
  --subscription "https://example.com/sub" \
  --desc "Personal subscription" \
  --interval 86400 \
  --auto-update

# Expected: Group created with subscription settings
# Prompt: "Update subscription now? [y/N]"
```

### Test 2.3: List Groups

```bash
./bin/raycoon group list

# Expected: Table showing:
# - ID
# - NAME
# - SUBSCRIPTION (Yes/No)
# - AUTO-UPDATE (✓/✗)
# - DESCRIPTION
```

### Test 2.4: Delete Group

```bash
# Try to delete global group (should fail)
./bin/raycoon group delete global
# Expected: Error - cannot delete global group

# Delete custom group
./bin/raycoon group delete work
# (press 'y' to confirm)

# Expected: Group and all its configs deleted
```

**✅ Group Management Test Results:**
- [ ] Create simple group
- [ ] Create group with subscription
- [ ] List groups
- [ ] Delete group
- [ ] Prevent deleting global group

---

## 3. Subscription Management

### Test 3.1: Check Subscription Status

```bash
./bin/raycoon sub status

# Expected: Table showing:
# - GROUP name
# - Number of CONFIGS
# - AUTO-UPDATE status
# - INTERVAL
# - LAST UPDATED time
# - NEXT UPDATE time
# - STATUS (OK / ⚠ Due)
```

### Test 3.2: Update Single Subscription

```bash
# This will fail with example.com, but tests the workflow
./bin/raycoon sub update personal

# Expected (on failure):
# Error message about fetch failure
# Retry attempts logged

# With real subscription URL:
# ✓ Subscription updated successfully!
# Total URIs: X
# Added: Y configs
# Failed: Z
```

### Test 3.3: Update All Subscriptions

```bash
./bin/raycoon sub update --all

# Expected:
# Updates all groups with subscriptions that are due
# Shows results for each group
```

### Test 3.4: Test with Mock Subscription

Create a mock subscription file:

```bash
# Create test subscription (base64 encoded proxy list)
cat > /tmp/test-sub.txt << 'EOF'
dmxlc3M6Ly9hMWIyYzNkNC0xMjM0LTU2NzgtOTBhYi1jZGVmMTIzNDU2NzhAZXhhbXBsZS5jb206NDQzP3R5cGU9d3Mmc2VjdXJpdHk9dGxzJnBhdGg9L3dzI1Rlc3QxCnZtZXNzOi8vZXlKMklqb2lNaUlzSW5Ceklqb2lWR1Z6ZERJaUxDSmhaR1FpT2lJeE1qY3VNQzR3TGpFaUxDSndiM0owSWpvaU9EQTRNQ0lzSW1sa0lqb2lZamd6TVRVeU1tUXRZelppTkMwME1qWmhMV0ptTXpRdFkyWmhaR1k0WVRkbE5EQTBJaXdpWVdsa0lqb2lNQ0lzSW5OamVTSTZJbUYxZEc4aUxDSnVaWFFpT2lKMFkzQWlMQ0owZVhCbElqb2libTl1WlNJc0ltaHZjM1FpT2lJaUxDSndZWFJvSWpvaUlpd2lkR3h6SWpvaUluMD0KdHJvamFuOi8vcGFzc3dvcmQxMjNAZXhhbXBsZS5jb206NDQzP3NlY3VyaXR5PXRscyZ0eXBlPXRjcCNUZXN0Mw==
EOF

# Start a simple HTTP server
python3 -m http.server 8000 &
HTTP_PID=$!

# Create group pointing to local server
./bin/raycoon group create test-local \
  --subscription "http://localhost:8000/test-sub.txt" \
  --interval 3600

# Update the subscription
./bin/raycoon sub update test-local

# Expected:
# ✓ Subscription updated successfully!
# Total URIs: 3
# Added: 3 configs

# Check the configs were added
./bin/raycoon config list --group test-local

# Cleanup
kill $HTTP_PID
```

**✅ Subscription Management Test Results:**
- [ ] Check subscription status
- [ ] Update single subscription (test error handling)
- [ ] Update all subscriptions
- [ ] Test with mock/local subscription
- [ ] Verify configs added from subscription
- [ ] Verify retry logic on failures

---

## 4. Database Operations

### Test 4.1: Verify Database Location

```bash
# Check database file
ls -lh ~/.local/share/raycoon/raycoon.db

# Expected: SQLite database file exists
```

### Test 4.2: Inspect Database

```bash
# Use sqlite3 to inspect (if installed)
sqlite3 ~/.local/share/raycoon/raycoon.db

# Run queries:
.tables
SELECT COUNT(*) FROM configs;
SELECT COUNT(*) FROM groups;
SELECT * FROM settings;
.quit
```

### Test 4.3: Test Transaction Safety

```bash
# Add multiple configs rapidly
for i in {1..5}; do
  ./bin/raycoon config add "vmess://..." --name "test-$i" &
done
wait

# List and verify all added
./bin/raycoon config list

# Expected: All configs added without corruption
```

**✅ Database Test Results:**
- [ ] Database file created
- [ ] Tables created properly
- [ ] Default settings inserted
- [ ] Global group exists
- [ ] Transactions work correctly

---

## 5. Connection Management

⚠️ **Requires Xray-core installed**

### Test 5.1: Check Xray Installation

```bash
# Check if xray is installed
which xray

# Check version
xray version

# If not installed:
# macOS: brew install xray
# Linux: Download from https://github.com/XTLS/Xray-core/releases
```

### Test 5.2: Connection Status (No Connection)

```bash
./bin/raycoon status

# Expected: "Status: Not connected"
```

### Test 5.3: Connect to Config

```bash
# Connect to config by ID
./bin/raycoon connect 1

# Expected (if Xray not installed):
# Error: xray binary not found

# Expected (if Xray installed):
# Connecting to test-server (vmess)...
# Address: 127.0.0.1:8080
# Mode: proxy
# SOCKS: 127.0.0.1:1080
# HTTP: 127.0.0.1:1081
# Core: xray
# ✓ Connected successfully!
```

### Test 5.4: Check Connection Status

```bash
./bin/raycoon status

# Expected:
# Connection Status
# ═════════════════
# Status: ● Connected
# Config: test-server (ID: 1)
# Protocol: vmess
# Address: 127.0.0.1:8080
# ...
```

### Test 5.5: Test Proxy

```bash
# Test SOCKS5 proxy (requires active connection)
curl --socks5 127.0.0.1:1080 https://ipinfo.io/json

# Test HTTP proxy
curl --proxy http://127.0.0.1:1081 https://ipinfo.io/json

# Expected: Your request goes through the proxy
```

### Test 5.6: Disconnect

```bash
./bin/raycoon disconnect

# Expected:
# Stopping proxy core...
# ✓ Disconnected successfully!
```

### Test 5.7: Connect with Custom Ports

```bash
./bin/raycoon connect 1 --port 2080 --http-port 2081

# Expected: Proxy running on custom ports
```

**✅ Connection Management Test Results:**
- [ ] Check Xray installation
- [ ] Status when not connected
- [ ] Connect to config
- [ ] Check status when connected
- [ ] Test proxy works (SOCKS5/HTTP)
- [ ] Disconnect
- [ ] Connect with custom ports

---

## 6. Error Handling

### Test 6.1: Invalid URI

```bash
./bin/raycoon config add "invalid://not-a-real-uri"

# Expected: Error message about invalid URI or unsupported protocol
```

### Test 6.2: Non-existent Config

```bash
./bin/raycoon config show 999999

# Expected: Error - config not found
```

### Test 6.3: Delete Non-existent Group

```bash
./bin/raycoon group delete nonexistent

# Expected: Error - group not found
```

### Test 6.4: Update Invalid Subscription

```bash
./bin/raycoon sub update invalid-group

# Expected: Error - group not found
```

### Test 6.5: Connect When Already Connected

```bash
# Connect to config 1
./bin/raycoon connect 1

# Try to connect to config 2 without disconnecting
./bin/raycoon connect 2

# Expected: Prompt to disconnect current connection
```

**✅ Error Handling Test Results:**
- [ ] Invalid URI handled
- [ ] Non-existent config handled
- [ ] Non-existent group handled
- [ ] Invalid subscription handled
- [ ] Already connected handled

---

## 7. Advanced Testing

### Test 7.1: Large Number of Configs

```bash
# Generate test script
cat > /tmp/test-many-configs.sh << 'EOF'
#!/bin/bash
for i in {1..100}; do
  ./bin/raycoon config add "vmess://..." --name "auto-test-$i"
done
EOF

chmod +x /tmp/test-many-configs.sh
/tmp/test-many-configs.sh

# List all configs
./bin/raycoon config list

# Expected: All 100 configs listed, performance acceptable
```

### Test 7.2: Concurrent Operations

```bash
# Test concurrent config additions
for i in {1..10}; do
  ./bin/raycoon config add "vmess://..." --name "concurrent-$i" &
done
wait

# Verify all added
./bin/raycoon config list | grep concurrent | wc -l
# Expected: 10
```

### Test 7.3: Database Cleanup

```bash
# Backup database
cp ~/.local/share/raycoon/raycoon.db ~/.local/share/raycoon/raycoon.db.backup

# Reset database
rm ~/.local/share/raycoon/raycoon.db

# Verify recreates on next run
./bin/raycoon group list

# Expected: Fresh database with only global group
```

**✅ Advanced Test Results:**
- [ ] Handle large number of configs
- [ ] Concurrent operations safe
- [ ] Database auto-creation works

---

## 8. Integration Testing

### Test 8.1: Complete Workflow

```bash
# 1. Create group with subscription
./bin/raycoon group create prod \
  --subscription "https://your-provider.com/sub" \
  --interval 86400

# 2. Update subscription
./bin/raycoon sub update prod

# 3. List configs from subscription
./bin/raycoon config list --group prod

# 4. Add manual config to global group
./bin/raycoon config add "vmess://..." --name "manual-backup"

# 5. Connect to a config
./bin/raycoon connect 1

# 6. Check status
./bin/raycoon status

# 7. Disconnect
./bin/raycoon disconnect

# 8. Check subscription status
./bin/raycoon sub status

# Expected: All operations complete successfully
```

**✅ Integration Test Results:**
- [ ] Complete workflow works end-to-end
- [ ] Multiple groups can coexist
- [ ] Manual and subscription configs work together
- [ ] All commands work in sequence

---

## Test Results Summary

### Quick Checklist

**Basic Operations:**
- [ ] Build successful (7.4MB binary)
- [ ] Database created on first run
- [ ] Help commands work
- [ ] Version command works

**Config Management:**
- [ ] Add VMess/VLESS/Trojan configs
- [ ] List with filters
- [ ] Show details
- [ ] Delete configs

**Group Management:**
- [ ] Create groups
- [ ] Create with subscriptions
- [ ] List groups
- [ ] Delete groups

**Subscription System:**
- [ ] Status command works
- [ ] Update command works
- [ ] Retry logic functions
- [ ] Configs imported correctly

**Connection (if Xray installed):**
- [ ] Connect to proxy
- [ ] Status shows details
- [ ] Proxy works (SOCKS5/HTTP)
- [ ] Disconnect works

**Error Handling:**
- [ ] Invalid inputs handled
- [ ] Network errors caught
- [ ] User-friendly messages

---

## Troubleshooting

### Issue: "xray binary not found"
**Solution:** Install Xray-core or specify path in settings

### Issue: "failed to fetch subscription"
**Solution:** Check internet connection, subscription URL validity

### Issue: "database locked"
**Solution:** Close other instances of raycoon

### Issue: "permission denied"
**Solution:** Check file permissions on database directory

---

## Next Steps

After testing current features:
1. Report any bugs found
2. Test with real proxy servers
3. Measure performance with large datasets
4. Test on different platforms (Linux, macOS, Windows)
5. Proceed to Phase 4: Latency Testing
