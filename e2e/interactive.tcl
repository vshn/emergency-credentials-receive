#!/usr/bin/env expect

source ./lib/common.tcl

set timeout 60

set cluster_id "c-appuio-lab-cloudscale-rma-0"
set api_endpoint "https://api.lab-cloudscale-rma-0.appuio.cloud:6443"

set passphrase [getenv_or_die "E2E_PASSBOLT_PASSPHRASE"]
set private_key [getenv_or_die "E2E_PASSBOLT_PRIVATE_KEY"]
set totp_key [getenv_or_die "E2E_PASSBOLT_TOTP_KEY_BASE32"]

proc expect_prompt {prompt} {
  expect -exact "$prompt"
  expect -exact "> "
  sleep .5
}

# The script assumes vi is used to enter the private key
set ::env(EDITOR) "vi"
file delete -force config.yaml
file delete -force "em-$cluster_id"
set ::env(EMR_CONFIG_DIR) [pwd]

log "Starting tool"
spawn ../emergency-credentials-receive -omit-token-output
expect -exact "Welcome"

log "Expecting private key prompt in editor"
expect -exact "Paste your Passbolt private key"
sleep .1
send -- "i"
log "Omitting user private_key input"
log_user 0
send -- "$private_key"
# Escape key
send -- "\x1b"
send -- ":x\r"
expect "survey*written"
log "private_key entry done"
log_user 1

log "Expecting passphrase prompt"
expect_prompt "Passbolt passphrase"
send -- "$passphrase"
send -- "\r"

log "Expecting cluster ID prompt"
expect_prompt "Enter your cluster ID"
send -- "$cluster_id"
send -- "\r"

log "Expecting TOTP prompt"
expect_prompt "Passbolt TOTP token"
send -- [totp_code_from_key $totp_key]
send -- "\r"

log "Expecting to have valid credentials"
expect -exact "2 buckets with credentials found"
expect -exact "Emergency credentials found"
expect -exact "OMITTED"

log "Expecting API endpoint prompt"
expect_prompt "Provide API endpoint"
send "$api_endpoint"
send -- "\r"
expect eof

test_kubeconfig "em-$cluster_id"

log "Test successful"

file delete -force config.yaml
file delete -force "em-$cluster_id"
