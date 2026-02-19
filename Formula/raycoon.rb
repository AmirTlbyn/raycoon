class Raycoon < Formula
  desc "Modern V2Ray/proxy CLI client with xray-core support"
  homepage "https://github.com/AmirTlbyn/raycoon"
  version "1.1.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/AmirTlbyn/raycoon/releases/download/v#{version}/raycoon-darwin-amd64"
      sha256 "1dd039b5d08217777e18d505756dd0d5d06099b196c817a80fb765fa5906c425"
    end
    on_arm do
      url "https://github.com/AmirTlbyn/raycoon/releases/download/v#{version}/raycoon-darwin-arm64"
      sha256 "ca5629771f97afff116842f51de54043ba1d3d302f628f5b3175ba57e0147752"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/AmirTlbyn/raycoon/releases/download/v#{version}/raycoon-linux-amd64"
      sha256 "1e91cd86591ac3b56e4614d261bde3190d2f05d7e5884e2cf7f1045552923414"
    end
    on_arm do
      url "https://github.com/AmirTlbyn/raycoon/releases/download/v#{version}/raycoon-linux-arm64"
      sha256 "0c135668f6dbe67f7b3f27ffac1449c0a8245760626d4b2447b34bfa9eaa2373"
    end
  end

  def install
    cpu = Hardware::CPU.arm? ? "arm64" : "amd64"
    os = OS.mac? ? "darwin" : "linux"
    bin.install "raycoon-#{os}-#{cpu}" => "raycoon"

    # Generate shell completions
    generate_completions_from_executable(bin/"raycoon", "completion")
  end

  def post_install
    # Create data directories
    (var/"raycoon").mkpath

    # Download xray-core and geo files
    xray_dir = Pathname.new(Dir.home)/".local"/"bin"
    xray_dir.mkpath

    unless (xray_dir/"xray").exist?
      ohai "Downloading xray-core..."
      xray_cpu = Hardware::CPU.arm? ? "arm64-v8a" : "64"
      xray_os = OS.mac? ? "macos" : "linux"
      xray_url = "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-#{xray_os}-#{xray_cpu}.zip"

      resource "xray" do
        url xray_url
      end

      resource("xray").stage do
        (xray_dir/"xray").write(Pathname.pwd/"xray")
        chmod 0755, xray_dir/"xray"
        cp "geoip.dat", xray_dir/"geoip.dat" if File.exist?("geoip.dat")
        cp "geosite.dat", xray_dir/"geosite.dat" if File.exist?("geosite.dat")
      end
    end
  end

  test do
    assert_match "Raycoon", shell_output("#{bin}/raycoon version")
  end
end
