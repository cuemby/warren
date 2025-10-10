# Homebrew Formula for Warren

This directory contains the Homebrew formula for Warren.

## For Users

### Install Warren via Homebrew

```bash
# Add Cuemby tap (once)
brew tap cuemby/tap

# Install Warren
brew install warren

# Verify installation
warren --version
```

### Update Warren

```bash
brew update
brew upgrade warren
```

### Uninstall Warren

```bash
brew uninstall warren
```

## For Maintainers

### Publishing a New Release

1. **Create GitHub Release**
   - Tag version (e.g., `v1.0.0`)
   - GitHub Actions automatically builds and uploads binaries

2. **Calculate SHA256 Hashes**
   ```bash
   # Download release artifacts
   curl -LO https://github.com/cuemby/warren/releases/download/v1.0.0/warren-darwin-arm64.tar.gz
   curl -LO https://github.com/cuemby/warren/releases/download/v1.0.0/warren-darwin-amd64.tar.gz
   curl -LO https://github.com/cuemby/warren/releases/download/v1.0.0/warren-linux-arm64.tar.gz
   curl -LO https://github.com/cuemby/warren/releases/download/v1.0.0/warren-linux-amd64.tar.gz

   # Calculate hashes
   shasum -a 256 *.tar.gz
   ```

3. **Update Formula**
   - Update `version` in `warren.rb`
   - Update all `url` fields with new version
   - Update all `sha256` fields with calculated hashes

4. **Test Formula Locally**
   ```bash
   # Audit formula
   brew audit --new-formula warren

   # Test installation
   brew install --build-from-source ./warren.rb

   # Test binary
   warren --version

   # Uninstall test
   brew uninstall warren
   ```

5. **Submit to Homebrew**

   **Option A: Homebrew Core (Recommended for popular projects)**
   ```bash
   # Fork homebrew-core
   git clone https://github.com/Homebrew/homebrew-core.git
   cd homebrew-core

   # Create formula
   cp /path/to/warren.rb Formula/warren.rb

   # Commit and push
   git checkout -b warren
   git add Formula/warren.rb
   git commit -m "warren 1.0.0 (new formula)"
   git push origin warren

   # Open PR on GitHub
   ```

   **Option B: Cuemby Tap (Easier, recommended for early releases)**
   ```bash
   # Create cuemby/homebrew-tap repository on GitHub

   # Clone tap
   git clone https://github.com/cuemby/homebrew-tap.git
   cd homebrew-tap

   # Add formula
   mkdir -p Formula
   cp /path/to/warren.rb Formula/warren.rb

   # Commit and push
   git add Formula/warren.rb
   git commit -m "Add Warren 1.0.0"
   git push

   # Users can now:
   # brew tap cuemby/tap
   # brew install warren
   ```

### Formula Maintenance

**Update existing formula:**
```bash
cd homebrew-tap
git pull

# Edit Formula/warren.rb
vim Formula/warren.rb

# Test
brew reinstall ./Formula/warren.rb

# Commit
git add Formula/warren.rb
git commit -m "warren: update to 1.1.0"
git push
```

**Deprecate formula:**
```ruby
# Add to warren.rb
deprecate! date: "2025-12-31", because: "reason"
```

### Troubleshooting

**Issue: SHA256 mismatch**
```
Error: SHA256 mismatch
Expected: abc123...
  Actual: def456...
```

Solution: Recalculate SHA256 from actual release artifacts

**Issue: Binary not found**
```
Error: No such file or directory
```

Solution: Check binary name in formula matches tarball contents

**Issue: Dependencies not found**
```
Error: containerd not found
```

Solution: Update `depends_on` in formula

## Resources

- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Homebrew Python Cookbook](https://docs.brew.sh/Python-for-Formula-Authors)
- [Homebrew Acceptable Formulae](https://docs.brew.sh/Acceptable-Formulae)
- [How to Open a Homebrew Pull Request](https://docs.brew.sh/How-To-Open-a-Homebrew-Pull-Request)

## Testing Checklist

Before submitting formula:

- [ ] Formula audits without errors (`brew audit`)
- [ ] Installation works (`brew install`)
- [ ] Binary runs (`warren --version`)
- [ ] Uninstallation works (`brew uninstall`)
- [ ] Reinstallation works (`brew reinstall`)
- [ ] Upgrade works (from previous version)
- [ ] Service works (if applicable)
- [ ] All platforms tested (macOS Intel, macOS ARM64)
