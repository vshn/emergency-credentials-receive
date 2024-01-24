
proc log {msg} {
  send_user "\n\[TEST\]\t$msg\n"
}

proc test_kubeconfig {kubeconfig} {
  log "Testing kubeconfig $kubeconfig"
  set ::env(KUBECONFIG) "$kubeconfig"

  log "Testing kubeconfig is allowed to get nodes"
  spawn kubectl get nodes
  expect -- "master*Ready*master"
  expect eof

  log "Testing kubeconfig is allowed to delete nodes"
  spawn kubectl auth can-i delete nodes
  expect -- "yes"
  expect eof
}

proc getenv_or_die {var} {
  if {![info exists ::env($var)]} {
    error "Missing environment variable $var"
  }
  return "$::env($var)"
}
