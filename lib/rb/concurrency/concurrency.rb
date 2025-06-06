# frozen_string_literal: true

require_relative "version"
require 'net/http'
require 'ffi'
require "json"

module Rb
  module Concurrency
    extend FFI::Library

    def self.so_path
      File.expand_path('../../../go/bin/arm-64/rb_concurrency.so', __dir__)
    end

    # TODO: golang のバイナリの so ファイルを起動して待機状態を作る
    ffi_lib so_path

    def self.free_go_string(ptr)
      FFI::MemoryPointer.from_address(ptr).free
    end

    attach_function :Proc, [:string], :string

    def self.process(reqs)
      result_str = Proc(reqs.to_json)

      pp result_str
    end
    
    class Error < StandardError; end
  end
end

reqs = [{
  "method" => "GET",
  "uri" => "http://localhost:8080",
  "headers" => {
    "Content-Type" => "application/json"
  },
}, {
  "method" => "GET",
  "uri" => "http://localhost:8080",
  "headers" => {
    "Content-Type" => "application/json"
  },
}]

Rb::Concurrency.process(reqs)