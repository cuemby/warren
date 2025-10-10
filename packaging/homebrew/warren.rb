# Homebrew Formula for Warren
# Usage: brew install cuemby/tap/warren

class Warren < Formula
  desc "Simple-yet-powerful container orchestrator for edge computing"
  homepage "https://github.com/cuemby/warren"
  version "1.0.0"

  # Update these URLs and SHA256 hashes after each release
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/cuemby/warren/releases/download/v1.0.0/warren-darwin-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    else
      url "https://github.com/cuemby/warren/releases/download/v1.0.0/warren-darwin-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
  elsif OS.linux?
    if Hardware::CPU.arm?
      url "https://github.com/cuemby/warren/releases/download/v1.0.0/warren-linux-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    else
      url "https://github.com/cuemby/warren/releases/download/v1.0.0/warren-linux-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
  end

  depends_on "containerd" => :recommended

  def install
    # Determine binary name based on platform
    if OS.mac?
      if Hardware::CPU.arm?
        bin.install "warren-darwin-arm64" => "warren"
      else
        bin.install "warren-darwin-amd64" => "warren"
      end
    elsif OS.linux?
      if Hardware::CPU.arm?
        bin.install "warren-linux-arm64" => "warren"
      else
        bin.install "warren-linux-amd64" => "warren"
      end
    end

    # Install shell completions
    generate_completions_from_executable(bin/"warren", "completion")

    # Install man pages (if available)
    # man1.install "docs/warren.1" if File.exist?("docs/warren.1")
  end

  def caveats
    <<~EOS
      Warren has been installed!

      To start a manager node:
        sudo warren cluster init

      To start a worker node:
        sudo warren worker start --manager <manager-ip>:8080

      For more information:
        warren --help
        https://github.com/cuemby/warren/docs

      Note: Warren requires containerd. Install with:
        brew install containerd
    EOS
  end

  service do
    run [opt_bin/"warren", "cluster", "init"]
    keep_alive true
    log_path var/"log/warren.log"
    error_log_path var/"log/warren-error.log"
  end

  test do
    system "#{bin}/warren", "--version"
    assert_match "warren version", shell_output("#{bin}/warren --version")
  end
end
