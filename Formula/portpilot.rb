class Portpilot < Formula
  desc "Local developer CLI for inspecting and controlling ports"
  homepage "https://github.com/GordenArcher/portpilot"
  url "https://github.com/GordenArcher/portpilot/archive/refs/tags/v0.1.1.tar.gz"
  sha256 "0b4dba35d2f49dc8745eb77534f23b4a930632b844725247830f0147e0899367"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-trimpath", "-ldflags=-s -w", "-o", bin/"portpilot", "."
  end

  test do
    assert_match "portpilot lets you scan", shell_output("#{bin}/portpilot --help")
  end
end
