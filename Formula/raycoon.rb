class Raycoon < Formula
  desc "Modern V2Ray/proxy CLI client with xray-core support"
  homepage "https://github.com/AmirTlbyn/raycoon"
  version "1.1.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/AmirTlbyn/raycoon/releases/download/v#{version}/raycoon-darwin-amd64"
      sha256 "28f52717da07cd620a8d7dd6159116abefaa522e376c33e9e9323b0f4b82df08"
    end
    on_arm do
      url "https://github.com/AmirTlbyn/raycoon/releases/download/v#{version}/raycoon-darwin-arm64"
      sha256 "76fa6b382592f73688e0764574583755adffdabfdc8fdbbd968e41639a46f614"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/AmirTlbyn/raycoon/releases/download/v#{version}/raycoon-linux-amd64"
      sha256 "b106bbc856e0f162948e5ad54825dc97f2af7e97c2237a5cc577c674aedd21e0"
    end
    on_arm do
      url "https://github.com/AmirTlbyn/raycoon/releases/download/v#{version}/raycoon-linux-arm64"
      sha256 "92c84ed5d0725db0f0f43832744af7c364a79baea685fb5ffd11b4eab3828072"
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
