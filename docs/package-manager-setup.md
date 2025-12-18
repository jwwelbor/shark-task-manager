# Package Manager Setup Guide

This guide explains how to set up the Homebrew tap and Scoop bucket repositories for the Shark Task Manager distribution system.

## Prerequisites

Task T-E04-F08-003 requires manual GitHub repository creation and token configuration. This document provides step-by-step instructions.

## Step 1: Create GitHub Repositories

### Create Homebrew Tap Repository

1. Go to https://github.com/new
2. Set the following:
   - **Owner**: jwwelbor
   - **Repository name**: `homebrew-shark` (MUST start with "homebrew-")
   - **Description**: "Homebrew tap for Shark Task Manager"
   - **Visibility**: Public (required for Homebrew)
   - **Initialize**: Do NOT add README, .gitignore, or license (GoReleaser will create Formula/)
3. Click "Create repository"

### Create Scoop Bucket Repository

1. Go to https://github.com/new
2. Set the following:
   - **Owner**: jwwelbor
   - **Repository name**: `scoop-shark` (MUST start with "scoop-")
   - **Description**: "Scoop bucket for Shark Task Manager"
   - **Visibility**: Public (required for Scoop)
   - **Initialize**: Do NOT add README, .gitignore, or license (GoReleaser will create bucket/)
3. Click "Create repository"

## Step 2: Create Fine-Grained Personal Access Tokens

### Create Homebrew Tap Token

1. Go to https://github.com/settings/tokens?type=beta
2. Click "Generate new token" (Fine-grained)
3. Configure the token:
   - **Token name**: `HOMEBREW_TAP_TOKEN`
   - **Expiration**: 90 days
   - **Description**: "Token for GoReleaser to publish Homebrew formulas"
   - **Repository access**: Select "Only select repositories"
     - Choose: `homebrew-shark`
   - **Permissions**:
     - Repository permissions:
       - Contents: **Read and write** access
       - Metadata: Read-only (automatically selected)
     - Disable all other permissions
4. Click "Generate token"
5. **IMPORTANT**: Copy the token immediately - you cannot view it again!
6. **Set calendar reminder** for 75 days from now to renew token

### Create Scoop Bucket Token

1. Go to https://github.com/settings/tokens?type=beta
2. Click "Generate new token" (Fine-grained)
3. Configure the token:
   - **Token name**: `SCOOP_BUCKET_TOKEN`
   - **Expiration**: 90 days
   - **Description**: "Token for GoReleaser to publish Scoop manifests"
   - **Repository access**: Select "Only select repositories"
     - Choose: `scoop-shark`
   - **Permissions**:
     - Repository permissions:
       - Contents: **Read and write** access
       - Metadata: Read-only (automatically selected)
     - Disable all other permissions
4. Click "Generate token"
5. **IMPORTANT**: Copy the token immediately - you cannot view it again!
6. **Set calendar reminder** for 75 days from now to renew token

## Step 3: Add Secrets to Main Repository

1. Go to https://github.com/jwwelbor/shark-task-manager/settings/secrets/actions
2. Click "New repository secret"

### Add HOMEBREW_TAP_TOKEN

1. **Name**: `HOMEBREW_TAP_TOKEN`
2. **Secret**: Paste the token from Step 2 (Homebrew)
3. Click "Add secret"

### Add SCOOP_BUCKET_TOKEN

1. **Name**: `SCOOP_BUCKET_TOKEN`
2. **Secret**: Paste the token from Step 2 (Scoop)
3. Click "Add secret"

## Step 4: Verify Configuration

After completing the manual steps above, verify the configuration:

### Test Token Permissions

Test that the tokens work with curl:

```bash
# Test Homebrew token (replace TOKEN with actual token)
curl -H "Authorization: token TOKEN" \
  https://api.github.com/repos/jwwelbor/homebrew-shark

# Test Scoop token (replace TOKEN with actual token)
curl -H "Authorization: token TOKEN" \
  https://api.github.com/repos/jwwelbor/scoop-shark
```

Both should return repository JSON (not 404 or 403).

### Validate GoReleaser Configuration

```bash
goreleaser check
```

Should pass with no errors. Note that it won't validate token access until actual release.

## Step 5: Update Workflow to Use Tokens

The GitHub Actions workflow needs to pass the tokens to GoReleaser. Update `.github/workflows/release.yml`:

```yaml
- name: Run GoReleaser
  uses: goreleaser/goreleaser-action@v6
  with:
    distribution: goreleaser
    version: '~> v2'
    args: release --clean
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
    SCOOP_BUCKET_TOKEN: ${{ secrets.SCOOP_BUCKET_TOKEN }}
```

## Step 6: Add READMEs to Package Manager Repositories

After the repositories are created, add README files:

### homebrew-shark/README.md

Use the template from `docs/homebrew-tap-README-template.md`

### scoop-shark/README.md

Use the template from `docs/scoop-bucket-README-template.md`

## Verification Checklist

After completing all steps:

- [ ] homebrew-shark repository created and public
- [ ] scoop-shark repository created and public
- [ ] HOMEBREW_TAP_TOKEN created with contents:write for homebrew-shark only
- [ ] SCOOP_BUCKET_TOKEN created with contents:write for scoop-shark only
- [ ] Both tokens added to shark-task-manager repository secrets
- [ ] Calendar reminders set for token renewal (75 days)
- [ ] GoReleaser configuration validated
- [ ] README files added to both package manager repositories
- [ ] Workflow updated to pass tokens to GoReleaser

## Token Renewal Process

When tokens expire (every 90 days):

1. Go to https://github.com/settings/tokens?type=beta
2. Find the expiring token
3. Click "Regenerate token"
4. Copy the new token
5. Update the secret in https://github.com/jwwelbor/shark-task-manager/settings/secrets/actions
6. Set new calendar reminder for 75 days from now

## Security Notes

- Tokens are scoped to specific repositories only (not all repos)
- Tokens have minimal permissions (contents:write only)
- Tokens expire after 90 days (security best practice)
- Tokens are stored as repository secrets (encrypted at rest)
- Tokens never appear in logs (GitHub automatically masks them)

## Troubleshooting

### "Repository not found" error during release

- Verify repositories are public
- Check repository names match exactly (homebrew-shark, scoop-shark)
- Verify tokens have access to correct repositories

### "Permission denied" error

- Verify tokens have contents:write permission
- Check tokens haven't expired
- Ensure secrets are named exactly: HOMEBREW_TAP_TOKEN, SCOOP_BUCKET_TOKEN

### Formula/manifest not published

- Check GitHub Actions logs for error messages
- Verify GoReleaser completed successfully
- Check package manager repository for commits from goreleaserbot

## Next Steps

After completing this setup:

1. Mark T-E04-F08-003 as complete
2. Proceed to T-E04-F08-004 (End-to-End Release Testing)
3. Test the complete release process with a beta tag
