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

require 'log-courier/event_queue'
require 'multi_json'
require 'thread'
require 'zlib'

class NativeException; end

module LogCourier
  class TimeoutError < StandardError; end
  class ShutdownSignal < StandardError; end
  class ProtocolError < StandardError; end

  class StreamFactory
    def create_stream()
      Stream.new
    end
  end

  class Stream
    def decode(event)
      yield event
    end
  end

  # Implementation of the server
  class Server
    attr_reader :port

    def initialize(options = {})
      @options = {
        logger:         nil,
        transport:      'tls',
        stream_factory: StreamFactory.new
      }.merge!(options)

      @logger = @options[:logger]
      @logger['plugin'] = 'input/courier'

      case @options[:transport]
      when 'tcp', 'tls'
        require 'log-courier/server_tcp'
        @server = ServerTcp.new(@options)
      when 'plainzmq', 'zmq'
        require 'log-courier/server_zmq'
        @server = ServerZmq.new(@options)
      else
        fail 'input/courier: \'transport\' must be tcp, tls, plainzmq or zmq'
      end

      # Grab the port back and update the logger context
      @port = @server.port
      @logger['port'] = @port unless @logger.nil?

      # Load the json adapter
      @json_adapter = MultiJson.adapter.instance
      @json_options = { raw: true, use_bigdecimal: true }
    end

    def run(&block)
      # TODO: Make queue size configurable
      event_queue = EventQueue.new 1
      server_thread = nil

      begin
        server_thread = Thread.new do
          # Receive messages and process them
          @server.run do |signature, message, comm|
            case signature
            when 'PING'
              process_ping message, comm
            when 'JDAT'
              process_jdat message, comm, event_queue
            else
              if comm.peer.nil?
                @logger.warn 'Unknown message received', :from => 'unknown' unless @logger.nil?
              else
                @logger.warn 'Unknown message received', :from => comm.peer unless @logger.nil?
              end
              # Don't kill a client that sends a bad message
              # Just reject it and let it send it again, potentially to another server
              comm.send '????', ''
            end
          end
        end

        loop do
          block.call event_queue.pop
        end
      ensure
        # Signal the server thread to stop
        unless server_thread.nil?
          server_thread.raise ShutdownSignal
          server_thread.join
        end
      end
      return
    end

    private

    def process_ping(message, comm)
      # Size of message should be 0
      if message.length != 0
        fail ProtocolError, "unexpected data attached to ping message (#{message.length})"
      end

      # PONG!
      # NOTE: comm.send can raise a Timeout::Error of its own
      comm.send 'PONG', ''
      return
    end

    def process_jdat(message, comm, event_queue)
      # Now we have the data, aim to respond within 5 seconds
      ack_timeout = Time.now.to_i + 5

      # OK - first is a nonce - we send this back with sequence acks
      # This allows the client to know what is being acknowledged
      # Nonce is 16 so check we have enough
      if message.length < 17
        fail ProtocolError, "JDAT message too small (#{message.length})"
      end

      nonce = message[0...16]

      if !@logger.nil? && @logger.debug?
        nonce_str = nonce.each_byte.map do |b|
          b.to_s(16).rjust(2, '0')
        end
      end

      # The remainder of the message is the compressed data block
      message = StringIO.new Zlib::Inflate.inflate(message[16...message.length])

      # Message now contains JSON encoded events
      # They are aligned as [length][event]... so on
      # We acknowledge them by their 1-index position in the stream
      # A 0 sequence acknowledgement means we haven't processed any yet
      sequence = 0
      events = []
      length_buf = ''
      data_buf = ''
      loop do
        ret = message.read 4, length_buf
        if ret.nil?
          # Finished!
          break
        elsif length_buf.length < 4
          fail ProtocolError, "JDAT length extraction failed (#{ret} #{length_buf.length})"
        end

        length = length_buf.unpack('N').first

        # Extract message
        ret = message.read length, data_buf
        if ret.nil? or data_buf.length < length
          @logger.warn()
          fail ProtocolError, "JDAT message extraction failed #{ret} #{data_buf.length}"
        end

        data_buf.force_encoding('utf-8')

        # Ensure valid encoding
        unless data_buf.valid_encoding?
          data_buf.chars.map do |c|
            c.valid_encoding? ? c : "\xEF\xBF\xBD"
          end
        end

        # Decode the JSON
        begin
          event = @json_adapter.load(data_buf, @json_options)
        rescue MultiJson::ParseError => e
          @logger.warn e, :hint => 'JSON parse failure, falling back to plain-text' unless @logger.nil?
          event = { 'message' => data_buf }
        end

        # Add peer fields?
        comm.add_fields event

        # Queue the event
        begin
          comm.stream.decode(event) do |event|
            event_queue.push event, [0, ack_timeout - Time.now.to_i].max
          end
        rescue TimeoutError
          # Full pipeline, partial ack
          # NOTE: comm.send can raise a Timeout::Error of its own
          @logger.debug 'Partially acknowledging message', :nonce => nonce_str.join, :sequence => sequence if !@logger.nil? && @logger.debug?
          comm.send 'ACKN', [nonce, sequence].pack('A*N')
          ack_timeout = Time.now.to_i + 5
          retry
        end

        sequence += 1
      end

      # Acknowledge the full message
      # NOTE: comm.send can raise a Timeout::Error
      @logger.debug 'Acknowledging message', :nonce => nonce_str.join, :sequence => sequence if !@logger.nil? && @logger.debug?
      comm.send 'ACKN', [nonce, sequence].pack('A*N')
      return
    end
  end
end
