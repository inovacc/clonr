# Security: Token Encryption with Keystore

Clonr uses a hierarchical key management system to protect OAuth tokens and sensitive data.

## Encryption Architecture

```
Root Secret (TPM-sealed or password-derived)
    │
    └── KEK (Key Encryption Key via HKDF)
            │
            └── Profile Master Keys (per-profile isolation)
                    │
                    └── DEKs (Data Encryption Keys per data type)
                            │
                            └── Encrypted Tokens (AES-256-GCM)
```

### Key Hierarchy

1. **Root Secret**: TPM-sealed when hardware is available, otherwise derived from machine-specific data
2. **KEK (Key Encryption Key)**: Derived from root secret using HKDF-SHA3-256
3. **Profile Master Keys**: Each profile has its own master key, encrypted with KEK
4. **DEKs (Data Encryption Keys)**: Per-profile, per-data-type keys for actual encryption

## Data Prefixes

Encrypted data is prefixed to indicate the encryption method:

| Prefix | Description |
|--------|-------------|
| `KS:`  | New keystore encryption (recommended) |
| `ENC:` | Legacy TPM encryption |
| `OPEN:` | Plain text (no encryption available) |

## Commands

### Rotate Keys

Rotate encryption keys for a profile without re-encrypting existing data:

```bash
clonr profile rotate <profile-name>
```

Key rotation:
- Generates a new profile master key
- Re-encrypts all DEKs with the new master key
- Existing encrypted data remains valid (DEKs are preserved)
- New encryptions use the rotated keys

### Migrate to Keystore

Migrate existing profiles from legacy encryption to keystore:

```bash
# Migrate a single profile
clonr profile migrate <profile-name>

# Migrate all profiles
clonr profile migrate --all

# Preview changes without migrating
clonr profile migrate --all --dry-run
```

Migration:
- Converts `OPEN:` and `ENC:` tokens to `KS:` format
- Requires TPM for `ENC:` token decryption
- Idempotent: already-migrated profiles are skipped

## Security Benefits

### Per-Profile Isolation

Each profile has its own encryption keys:
- Compromise of one profile's keys doesn't affect others
- Keys can be rotated independently
- Clean deletion: removing a profile removes its keys

### TPM Integration

When a TPM is available:
- Root secret is sealed to the TPM
- Cannot be extracted without TPM access
- Tied to the specific machine

### Key Rotation

Regular key rotation limits exposure:
- Old keys are securely replaced
- Forward secrecy for new data
- No re-encryption of existing data needed

## Storage

Keystore data is stored at:
- **Linux/macOS**: `~/.config/clonr/.clonr_keystore/`
- **Windows**: `%APPDATA%\clonr\.clonr_keystore\`

Files:
- `keyring.enc` - Encrypted profile master keys
- `<profile>/envelope.enc` - Encrypted DEKs per profile

## Best Practices

1. **Use TPM when available**: Provides hardware-backed security
2. **Rotate keys periodically**: `clonr profile rotate <name>`
3. **Migrate legacy profiles**: `clonr profile migrate --all`
4. **Backup encrypted exports**: Use `clonr data export` for backups
