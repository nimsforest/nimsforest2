# Hetzner Storage Box Setup for Release Binary Distribution

This guide explains how to set up a Hetzner Storage Box to host downloadable release binaries for NimsForest.

## Overview

When configured, the release workflow will automatically upload built binaries to your Hetzner Storage Box, making them available for public download via HTTPS (WebDAV).

## Prerequisites

- A Hetzner Storage Box (any size - BX10 or larger recommended)
- GitHub repository admin access to configure secrets

## Step 1: Order a Hetzner Storage Box

1. Go to [Hetzner Storage Box](https://www.hetzner.com/storage/storage-box)
2. Choose a plan (BX10 with 100GB is sufficient for most projects)
3. Complete the order

## Step 2: Configure Storage Box for WebDAV Access

After receiving your storage box credentials:

1. Log into [Hetzner Robot](https://robot.hetzner.com)
2. Navigate to **Storage Box** → Select your storage box
3. Go to **Settings**
4. Enable the following services:
   - ✅ **SSH/SFTP** (for uploads)
   - ✅ **WebDAV** (for public downloads via HTTPS)
   - ✅ **External reachability** (if you want public access)

### Configure Sub-account (Recommended)

For better security, create a sub-account with limited permissions:

1. In Robot, go to **Storage Box** → **Sub-accounts**
2. Click **Create sub-account**
3. Settings:
   - **Username**: e.g., `releases`
   - **Home directory**: `/releases`
   - **SSH/SFTP**: ✅ Enabled
   - **WebDAV**: ✅ Enabled (read-only for public access)
   - **External reachability**: ✅ Enabled
4. Note the generated password or set your own

## Step 3: Set Up SSH Key Authentication (Recommended)

For more secure CI/CD authentication:

```bash
# Generate a dedicated SSH key
ssh-keygen -t ed25519 -f ~/.ssh/storagebox_deploy -N "" -C "github-actions-deploy"

# Display the public key to add to storage box
cat ~/.ssh/storagebox_deploy.pub
```

Add the public key to your storage box:

1. In Robot, go to **Storage Box** → **SSH Keys**
2. Add your public key
3. Or via command line:
   ```bash
   echo "your-public-key" | ssh -p 23 u123456@u123456.your-storagebox.de "mkdir -p .ssh && cat >> .ssh/authorized_keys"
   ```

## Step 4: Configure GitHub Repository

### Add Repository Variables

Go to your GitHub repository → **Settings** → **Secrets and variables** → **Actions** → **Variables**:

| Variable | Value | Description |
|----------|-------|-------------|
| `ENABLE_STORAGE_BOX_PUBLISH` | `true` | Enable storage box uploads |

### Add Repository Secrets

Go to **Settings** → **Secrets and variables** → **Actions** → **Secrets**:

| Secret | Value | Description |
|--------|-------|-------------|
| `STORAGEBOX_HOST` | `u123456.your-storagebox.de` | Your storage box hostname |
| `STORAGEBOX_USER` | `u123456` or `u123456-releases` | Username (main or sub-account) |
| `STORAGEBOX_PASSWORD` | `your-password` | Password (if not using SSH key) |
| `STORAGEBOX_SSH_KEY` | `-----BEGIN OPENSSH...` | Private SSH key (recommended) |

**Note**: Use either `STORAGEBOX_PASSWORD` OR `STORAGEBOX_SSH_KEY`, not both. SSH key is recommended.

## Step 5: Test the Configuration

1. Create a test tag to trigger a release:
   ```bash
   git tag v0.0.1-test
   git push origin v0.0.1-test
   ```

2. Check the GitHub Actions workflow for the release

3. Verify files are uploaded:
   ```bash
   # Via SFTP
   sftp -P 23 u123456@u123456.your-storagebox.de
   ls releases/
   
   # Or via WebDAV (browser)
   # https://u123456.your-storagebox.de/releases/
   ```

## Directory Structure

After releases, your storage box will have this structure:

```
/releases/
├── v1.0.0/
│   ├── forest-linux-amd64.tar.gz
│   ├── forest-linux-amd64.sha256
│   ├── forest-linux-arm64.tar.gz
│   ├── forest-linux-arm64.sha256
│   ├── forest-darwin-amd64.tar.gz
│   ├── forest-darwin-amd64.sha256
│   ├── forest-darwin-arm64.tar.gz
│   ├── forest-darwin-arm64.sha256
│   └── index.html
├── v1.0.1/
│   └── ...
└── latest/
    └── ... (copy of most recent release)
```

## Public Download URLs

Once configured, your releases will be available at:

```
https://u123456.your-storagebox.de/releases/latest/forest-linux-amd64.tar.gz
https://u123456.your-storagebox.de/releases/v1.0.0/forest-linux-amd64.tar.gz
```

### Custom Domain (Optional)

You can point a custom domain to your storage box:

1. Create a CNAME record: `downloads.yourdomain.com` → `u123456.your-storagebox.de`
2. Or use a reverse proxy with SSL (recommended for custom domains)

## Troubleshooting

### Upload fails with "Permission denied"

- Check that SSH/SFTP is enabled for your user
- Verify the credentials are correct
- If using sub-account, ensure home directory exists

### WebDAV returns 403 Forbidden

- Enable "External reachability" in storage box settings
- Check that WebDAV is enabled for your user
- Ensure the `/releases` directory exists

### SSH key authentication fails

- Verify the key is in OpenSSH format
- Check key permissions (600 for private key)
- Ensure public key is added to storage box `~/.ssh/authorized_keys`

### Check connection manually

```bash
# Test SFTP connection
sftp -P 23 u123456@u123456.your-storagebox.de

# Test with verbose output
sftp -v -P 23 u123456@u123456.your-storagebox.de
```

## Security Considerations

1. **Use SSH keys** instead of passwords when possible
2. **Use a sub-account** with limited permissions for CI/CD
3. **Enable read-only WebDAV** for public downloads if supported
4. **Monitor access logs** in Hetzner Robot

## Cost

Hetzner Storage Box pricing (as of 2024):
- BX10 (100 GB): ~€3.29/month
- BX20 (500 GB): ~€5.83/month
- BX30 (1 TB): ~€10.59/month

This is a cost-effective solution for hosting release binaries.

## Alternative: GitHub Releases Only

If you don't want to set up a storage box, the release workflow will still upload binaries to GitHub Releases. The storage box upload is optional and controlled by the `ENABLE_STORAGE_BOX_PUBLISH` variable.
