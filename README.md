# emergency-credentials-receive

Guided wizard to receive and decrypt cluster emergency credentials from VSHN Passbolt and our emergency credentials buckets.

## Usage

```sh
# Interactive mode
emergency-credentials-receive

# Non-interactive mode (after private key setup)
EMR_PASSPHRASE=$(pass vshn/passbolt) emergency-credentials-receive c-crashy-wreck-1234
```

## Install from binary

Install the latest release for your arch and OS with the following command:

```sh
curl -s "https://raw.githubusercontent.com/vshn/emergency-credentials-receive/main/install.sh" | bash

# Guided setup
emergency-credentials-receive
```

## Development

There are E2E tests in the `e2e` directory.
They simulate user inputs using [expect](https://core.tcl-lang.org/expect/index).

To run the tests you need your passbolt private key and the passbolt passphrase.
Or you can use the test credentials from [git.vshn.net](https://git.vshn.net/syn/passbolt-pubkey-sync/-/settings/ci_cd).
Note that the test credentials can only access a very limited set of test clusters.
You can set them as environment variables:

```sh
export E2E_PASSBOLT_PASSPHRASE="..."
export E2E_PASSBOLT_PRIVATE_KEY="$(cat /path/to/private.key)"
```
