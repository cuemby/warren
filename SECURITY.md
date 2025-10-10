# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

**Note**: We currently support the latest major version (1.x). Security updates are backported to the latest minor release only.

---

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

We take security seriously and appreciate your efforts to responsibly disclose your findings. To report a security vulnerability, please follow these steps:

### 1. Contact Us Privately

Send an email to **security@cuemby.com** with the following information:

- **Subject Line**: Include "WARREN SECURITY" in the subject
- **Description**: A clear description of the vulnerability
- **Impact**: How the vulnerability can be exploited and its potential impact
- **Reproduction Steps**: Detailed steps to reproduce the issue
- **Affected Versions**: Which versions of Warren are affected
- **Suggested Fix**: If you have a fix or mitigation suggestion (optional)
- **Your Contact Information**: How we can reach you for follow-up

### 2. What to Expect

- **Acknowledgment**: We will acknowledge receipt of your report within **48 hours**
- **Initial Assessment**: We will provide an initial assessment within **5 business days**
- **Status Updates**: We will keep you informed of our progress every **7 days** until resolution
- **Resolution Timeline**: We aim to resolve critical vulnerabilities within **30 days**

### 3. Coordinated Disclosure

We follow a **coordinated disclosure** process:

1. We will work with you to understand and verify the vulnerability
2. We will develop and test a fix
3. We will prepare a security advisory
4. We will release the fix and publish the advisory
5. We will credit you (unless you prefer to remain anonymous)

**Embargo Period**: We request a **90-day embargo** before public disclosure to allow users time to update.

---

## Security Update Process

### How We Handle Security Issues

1. **Triage**: Assess severity using CVSS v3.1 scoring
2. **Fix Development**: Develop and test a fix internally
3. **Security Advisory**: Prepare GitHub Security Advisory (GHSA)
4. **Release**: Release patched versions for supported releases
5. **Notification**: Notify users via:
   - GitHub Security Advisory
   - Release notes
   - Mailing list (if applicable)
   - Social media channels

### Severity Levels

| Severity | CVSS Score | Response Time | Example |
|----------|------------|---------------|---------|
| **Critical** | 9.0-10.0 | 24-48 hours | Remote code execution, authentication bypass |
| **High** | 7.0-8.9 | 1 week | Privilege escalation, data exposure |
| **Medium** | 4.0-6.9 | 2-4 weeks | Denial of service, information disclosure |
| **Low** | 0.1-3.9 | Next regular release | Minor information leaks |

---

## Security Best Practices

### For Warren Users

1. **Keep Warren Updated**: Always run the latest version
   ```bash
   # Check your version
   warren --version

   # Update via package manager
   brew upgrade warren  # macOS
   apt update && apt upgrade warren  # Ubuntu
   ```

2. **Use TLS/mTLS**: Enable mTLS for API communication
   ```bash
   warren cluster init --enable-mtls
   ```

3. **Secure Join Tokens**: Protect manager and worker join tokens
   ```bash
   # Tokens expire after 24 hours by default
   # Generate new tokens regularly
   warren cluster join-token manager
   ```

4. **Encrypt Secrets**: Secrets are encrypted at rest by default (AES-256-GCM)
   ```bash
   warren secret create db-password --from-literal password=secret
   ```

5. **Network Isolation**: Use WireGuard for encrypted overlay networking
   - All inter-node traffic is encrypted by default

6. **Access Control**: Limit who can access manager nodes
   - Run managers in private networks
   - Use firewall rules to restrict API access

7. **Monitoring**: Enable metrics and logging
   ```bash
   warren cluster init --metrics-addr 127.0.0.1:9090 --log-level info
   ```

8. **Regular Audits**: Review cluster configuration and access logs

### For Warren Developers

1. **Input Validation**: Always validate user input
2. **Error Handling**: Don't leak sensitive information in errors
3. **Authentication**: Verify tokens before accepting join requests
4. **Authorization**: Check permissions before state changes
5. **Cryptography**: Use standard libraries (AES-256-GCM, mTLS)
6. **Dependencies**: Keep dependencies updated and scan for vulnerabilities
7. **Code Review**: All security-related changes require review
8. **Testing**: Include security test cases

---

## Known Security Considerations

### Current Security Features

- ✅ **Secrets Encryption**: AES-256-GCM encryption at rest
- ✅ **WireGuard Networking**: Encrypted overlay networking
- ✅ **Join Tokens**: Time-limited tokens (24h) for cluster joining
- ✅ **API Authentication**: Token-based authentication for manager API
- ✅ **Raft Security**: Secure consensus with authenticated peers

### Future Security Enhancements (Roadmap)

- 🔜 **mTLS for API**: Mutual TLS for all API communication (M6)
- 🔜 **RBAC**: Role-based access control (M6)
- 🔜 **Audit Logging**: Comprehensive audit trail (M6)
- 🔜 **Image Scanning**: Automatic vulnerability scanning for container images (M7)
- 🔜 **Network Policies**: Fine-grained network access control (M7)
- 🔜 **Pod Security Standards**: Security contexts for containers (M7)

### Known Limitations (Pre-1.0)

- **No RBAC**: All authenticated users have full cluster access
- **No mTLS by default**: API uses token auth, not mTLS (opt-in)
- **No image verification**: Container images are not verified (signature checking not implemented)
- **No network policies**: All containers can communicate (WireGuard encryption only)

**Recommendation**: Run Warren in trusted networks until v1.0 with full security features.

---

## Security Hall of Fame

We would like to thank the following individuals/organizations for responsibly disclosing security vulnerabilities:

<!-- This section will be populated as vulnerabilities are reported and fixed -->

*No security vulnerabilities have been reported yet.*

---

## GPG Key for Secure Communication

For sensitive security reports, you may encrypt your email using our GPG key:

```
-----BEGIN PGP PUBLIC KEY BLOCK-----
[GPG key will be added here]
-----END PGP PUBLIC KEY BLOCK-----
```

**Key Fingerprint**: [Will be added]

---

## Security Resources

- **GitHub Security Advisories**: https://github.com/cuemby/warren/security/advisories
- **CVE Database**: https://cve.mitre.org/
- **Go Security**: https://go.dev/security/
- **OWASP Guidelines**: https://owasp.org/

---

## Contact

- **Security Issues**: security@cuemby.com
- **General Questions**: opensource@cuemby.com
- **Bug Reports** (non-security): [GitHub Issues](https://github.com/cuemby/warren/issues)

---

**Thank you for helping keep Warren and the community safe!** 🔒
