# encoding: utf-8

# Copyright 2014 Jason Woods.
#
# This file is a modification of code from Logstash Forwarder.
# Copyright 2012-2013 Jordan Sissel and contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

require 'cabin'
require 'log-courier/cabin-copy' # Used by the plugin so must be required here
require 'log-courier/event_queue'
require 'thread'

class NativeException; end

module LogCourier
  class TimeoutError < StandardError; end
  class ShutdownSignal < StandardError; end
  class ProtocolError < StandardError; end

  # Implementation of the server
  class Server
    attr_reader :port

    # TODO(driskell): Consolidate singleton into another file
    class << self
      @json_adapter
      @json_parseerror

      def get_json_adapter
        @json_adapter = MultiJson.adapter.instance if @json_adapter.nil?
        @json_adapter
      end

      def get_json_parseerror
        if @json_parseerror.nil?
          @json_parseerror = get_json_adapter.class::ParseError
        end
        @json_parseerror
      end
    end

    def initialize(options = {})
      @options = {
        transport: 'tls',
      }.merge!(options)

      if @options[:logger]
        @logger = @options[:logger].fork(LogCourier)
      else
        @logger = Cabin::Channel.new
      end

      case @options[:transport]
      when 'tcp', 'tls'
        require 'log-courier/server_tcp'
        @server = TCPTransport.new(@options)
      else
        fail 'input/courier: \'transport\' must be tcp or tls'
      end

      # Grab the port back and update the logger context
      @port = @server.port
      @logger['port'] = @port unless @logger.nil?
    end

    def run(&block)
      # TODO: Make queue size configurable
      event_queue = EventQueue.new 1

      begin
        @server.run event_queue

        loop do
          block.call event_queue.pop
        end
      rescue => e
        puts e.inspect, e.backtrace
      ensure
        # Signal the server thread to stop
        @server.shutdown
      end

      return
    end
  end
end
