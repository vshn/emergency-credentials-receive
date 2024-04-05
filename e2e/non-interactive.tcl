#!/usr/bin/env expect

source ./lib/common.tcl

set timeout 60

set cluster_id "c-appuio-lab-cloudscale-rma-0"
set api_endpoint "https://api.lab-cloudscale-rma-0.appuio.cloud:6443"

set passphrase [getenv_or_die "E2E_PASSBOLT_PASSPHRASE"]
set private_key [getenv_or_die "E2E_PASSBOLT_PRIVATE_KEY"]
set totp_key [getenv_or_die "E2E_PASSBOLT_TOTP_KEY_BASE32"]

file delete -force config.yaml
file delete -force "em-$cluster_id"
set ::env(EMR_CONFIG_DIR) [pwd]

log "Starting tool"
set ::env(EMR_PASSPHRASE) "$passphrase"
set ::env(EMR_TOTP_TOKEN) [totp_code_from_key $totp_key]
set ::env(EMR_KUBERNETES_ENDPOINT) "$api_endpoint"
exec -- jq --null-input --arg key "$private_key" {{passbolt_key: $key}} > config.yaml
spawn ../emergency-credentials-receive -omit-token-output "$cluster_id"
expect -exact "Welcome"

log "Expecting to have valid credentials"
expect -exact "2 buckets with credentials found"
expect -exact "Emergency credentials found"
expect -exact "OMITTED"
expect eof

test_kubeconfig "em-$cluster_id"

log "Test successful"

file delete -force config.yaml
file delete -force "em-$cluster_id"
