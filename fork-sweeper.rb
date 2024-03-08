# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class ForkSweeper < Formula
  desc "Remove unused GitHub forks"
  homepage "https://github.com/rednafi/fork-sweeper"
  version "0.1.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/rednafi/fork-sweeper/releases/download/v0.1.3/fork-sweeper_Darwin_arm64.tar.gz"
      sha256 "285f5f588fe0776f2a052261c9895f885495456195ad73f005482bb012519ad9"

      def install
        bin.install "fork-sweeper"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/rednafi/fork-sweeper/releases/download/v0.1.3/fork-sweeper_Darwin_x86_64.tar.gz"
      sha256 "c41ffaeb22b78ce2e15403b9ed672c73194f273a91a2463383179ac21842142d"

      def install
        bin.install "fork-sweeper"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/rednafi/fork-sweeper/releases/download/v0.1.3/fork-sweeper_Linux_arm64.tar.gz"
      sha256 "58fbda0bec5b8d15a7d2e11fbc4f8f619b6b9e4c329912ad60dda06a6ff61db4"

      def install
        bin.install "fork-sweeper"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/rednafi/fork-sweeper/releases/download/v0.1.3/fork-sweeper_Linux_x86_64.tar.gz"
      sha256 "ab67f17504d50f37d276e8ebf149dd0e41470da36ca44523aa7a54c409d958cc"

      def install
        bin.install "fork-sweeper"
      end
    end
  end
end
