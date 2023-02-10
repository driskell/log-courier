# Copyright 2014-2021 Jason Woods and Contributors.
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
require 'log-courier/protocol'
require 'log-courier/version'
require 'multi_json'
require 'zlib'

module LogCourier
  class TimeoutError < StandardError; end

  class ShutdownSignal < StandardError; end

  class ProtocolError < StandardError; end

  # Implementation of the server
  class Server
    attr_reader :port

    def initialize(options = {})
      @options = {
        logger: nil,
        transport: 'tls',
        disable_handshake: false,
      }.merge!(options)

      @logger = @options[:logger]

      case @options[:transport]
      when 'tcp', 'tls'
        require 'log-courier/server_tcp'
        @server = ServerTcp.new(@options)
      else
        raise 'input/courier: \'transport\' must be tcp or tls'
      end

      # Grab the port back and update the logger context
      @port = @server.port

      # TODO: Make queue size configurable
      @event_queue = EventQueue.new 1

      @server_thread = Thread.new do
        # Receive messages and process them
        @server.run do |signature, message, comm|
          case signature
          when 'PING'
            process_ping message, comm
          when 'JDAT'
            process_jdat message, comm, @event_queue
          else
            if comm.peer.nil?
              @logger&.warn 'Unknown message received', from: 'unknown'
            else
              @logger&.warn 'Unknown message received', from: comm.peer
            end
            # Don't kill a client that sends a bad message
            # Just reject it and let it send it again, potentially to another server
            comm.send '????', ''
          end
        end
      end
    end

    def run(&block)
      loop do
        event = @event_queue.pop
        break if event.nil?

        block.call event
      end
      nil
    end

    def stop
      @server_thread.raise ShutdownSignal
      @event_queue << nil
      @server_thread.join
      nil
    end

    private

    def process_ping(message, comm)
      # Size of message should be 0
      raise ProtocolError, "unexpected data attached to ping message (#{message.bytesize})" unless message.bytesize.zero?

      # PONG!
      # NOTE: comm.send can raise a Timeout::Error of its own
      comm.send 'PONG', ''
    end

    def process_jdat(message, comm, event_queue)
      # Now we have the data, aim to respond within 5 seconds
      ack_timeout = Time.now.to_i + 5

      # OK - first is a nonce - we send this back with sequence acks
      # This allows the client to know what is being acknowledged
      # Nonce is 16 so check we have enough
      raise ProtocolError, "JDAT message too small (#{message.bytesize})" if message.bytesize < 17

      nonce = message[0...16]

      if @logger&.debug?
        nonce_str = nonce.each_byte.map do |b|
          b.to_s(16).rjust(2, '0')
        end
      end

      # The remainder of the message is the compressed data block
      message = StringIO.new Zlib::Inflate.inflate(message.byteslice(16, message.bytesize))

      # Message now contains JSON encoded events
      # They are aligned as [length][event]... so on
      # We acknowledge them by their 1-index position in the stream
      # A 0 sequence acknowledgement means we haven't processed any yet
      sequence = 0
      length_buf = ''
      data_buf = ''
      loop do
        ret = message.read 4, length_buf
        # Finished?
        break if ret.nil?
        raise ProtocolError, "JDAT length extraction failed (#{ret} #{length_buf.bytesize})" if length_buf.bytesize < 4

        length = length_buf.unpack1('N')

        # Extract message
        ret = message.read length, data_buf
        if ret.nil? || data_buf.bytesize < length
          @logger&.warn()
          raise ProtocolError, "JDAT message extraction failed #{ret} #{data_buf.bytesize}"
        end

        data_buf.force_encoding('utf-8')

        # Ensure valid encoding
        invalid_encodings = 0
        unless data_buf.valid_encoding?
          data_buf.chars.map do |c|
            if c.valid_encoding?
              c
            else
              invalid_encodings += 1
              "\xEF\xBF\xBD"
            end
          end
        end

        # Decode the JSON
        begin
          event = MultiJson.load(data_buf)
        rescue MultiJson::ParseError => e
          @logger&.warn e, invalid_encodings: invalid_encodings, hint: 'JSON parse failure, falling back to plain-text'
          event = { 'message' => data_buf }
        end

        # Add peer fields?
        comm.add_fields event

        # Queue the event
        begin
          event_queue.push event, [0, ack_timeout - Time.now.to_i].max
        rescue TimeoutError
          # Full pipeline, partial ack
          # NOTE: comm.send can raise a Timeout::Error of its own
          @logger&.debug 'Partially acknowledging message', nonce: nonce_str.join, sequence: sequence if @logger&.debug?
          comm.send 'ACKN', [nonce, sequence].pack('a*N')
          ack_timeout = Time.now.to_i + 5
          retry
        end

        sequence += 1
      end

      # Acknowledge the full message
      # NOTE: comm.send can raise a Timeout::Error
      @logger&.debug 'Acknowledging message', nonce: nonce_str.join, sequence: sequence if @logger&.debug?
      comm.send 'ACKN', [nonce, sequence].pack('A*N')
    end
  end
end
