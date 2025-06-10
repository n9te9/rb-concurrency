# frozen_string_literal: true

require_relative "version"
require 'net/http'
require 'ffi'
require "json"

module Rb
  module Concurrency
    extend FFI::Library

    CPU_ARCH_MAP = {
      'x86_64' => 'amd64',
      'arm64'  => 'arm64',
      'aarch64' => 'arm64'
    }.freeze

    def self.current_go_arch
      ruby_cpu = Gem::Platform.local.cpu
      ruby_os = Gem::Platform.local.os
      mapped_arch = CPU_ARCH_MAP[ruby_cpu]

      unless mapped_arch
        raise Error, "Unsupported CPU architecture: #{ruby_cpu}. Cannot determine Go binary path."
      end
      "#{ruby_os}-#{mapped_arch}"
    end

    def self.so_path
      File.expand_path("../../../../go/bin/#{self.current_go_arch}/rb_concurrency.so", __dir__)
    end

    ffi_lib so_path

    attach_function :Proc, [:string], :string

    # Process the requests concurrently.
    # @param requests [Array<Request>]
    # @return [Array<Response>]
    def self.process(requests)
      requests_hash = requests.map do |req|
        if req.is_a?(Request)
          req.to_h
        elsif req.is_a?(Hash)
          req
        else
          raise Error, "Invalid request type: #{req.class}. Expected Request or Hash."
        end
      end

      result_str = Proc(requests_hash.to_json)
      results = JSON.parse(result_str)

      results.map do |result|
        if result.is_a?(Hash)
          Response.from_json(result.to_json)
        elsif result.is_a?(String)
          result_hash = JSON.parse(result)
          Response.new(
            status: result_hash['status'],
            headers: result_hash['headers'] || {},
            body: result_hash['body']
          )
        else
          raise Error, "Invalid response format: #{result.inspect}"
        end
      end
    end

    class Request
      attr_accessor :method, :uri, :headers, :body

      def initialize(method:, uri:, headers: {}, body: nil)
        @method = method
        @uri = uri
        @headers = headers
        @body = body
      end

      def to_h
        {
          "method" => @method,
          "uri" => @uri,
          "headers" => @headers,
          "body" => @body
        }
      end

      def to_json(*_args)
        to_h.to_json
      end
    end

    class Response
      attr_accessor :status, :headers, :body
      def initialize(status:, headers: {}, body: nil)
        @status = status
        @headers = headers
        @body = body
      end

      def to_h
        {
          "status" => @status,
          "headers" => @headers,
          "body" => @body
        }
      end

      def to_json(*_args)
        to_h.to_json
      end

      def self.from_json(json_str)
        data = JSON.parse(json_str)
        new(
          status: data['status'],
          headers: data['headers'] || {},
          body: data['body']
        )
      end
    end
    
    class Error < StandardError; end
  end
end

# requests = [
#   Rb::Concurrency::Request::Request.new(method: 'GET', uri: 'http://localhost:8181?count=1', headers: { 'Accept' => ['application/json'] }),
#   Rb::Concurrency::Request::Request.new(method: 'GET', uri: 'http://localhost:8181?count=2', headers: { 'Accept' => ['application/json'] }),
#   Rb::Concurrency::Request::Request.new(method: 'GET', uri: 'http://localhost:8181?count=3', headers: { 'Accept' => ['application/json'] }),
#   Rb::Concurrency::Request::Request.new(method: 'GET', uri: 'http://localhost:8181?count=4', headers: { 'Accept' => ['application/json'] }),
#   Rb::Concurrency::Request::Request.new(method: 'GET', uri: 'http://localhost:8181?count=5', headers: { 'Accept' => ['application/json'] }),
#   Rb::Concurrency::Request::Request.new(method: 'GET', uri: 'http://localhost:8181?count=6', headers: { 'Accept' => ['application/json'] }),
# ]

# responses = Rb::Concurrency::Request.process(requests)
# responses.each do |response|
#   puts "Status: #{response.status}"
#   puts "Headers: #{response.headers}"
#   puts "Body: #{response.body}"
#   puts "-------------------"
# end