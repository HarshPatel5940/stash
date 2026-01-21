# Homebrew Formula for Stash
# This file is a template - actual formula will be in homebrew-stash tap

class Stash < Formula
  desc "Mac backup CLI tool - stash your Mac, restore anywhere"
  homepage "https://github.com/harshpatel5940/stash"
  url "https://github.com/harshpatel5940/stash/archive/refs/tags/v1.2.0.tar.gz"
  sha256 "REPLACE_WITH_ACTUAL_SHA256"
  license "MIT"
  version "1.2.0"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w")

    # Generate shell completions
    generate_completions_from_executable(bin/"stash", "completion")
  end

  def caveats
    <<~EOS
      Stash has been installed!

      To get started:
        1. Initialize stash: stash init
        2. Preview backup: stash list
        3. Create backup: stash backup

      Your encryption key will be stored at ~/.stash.key
      Keep this key safe - you'll need it to restore backups!

      Configuration: ~/.stash.yaml
      Default backup location: ~/stash-backups/

      For more information: https://github.com/harshpatel5940/stash
    EOS
  end

  test do
    assert_match "stash version", shell_output("#{bin}/stash --version")
  end
end
