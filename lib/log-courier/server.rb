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

module LogCourier
  class TimeoutError < StandardError; end
  class ShutdownSignal < StandardError; end
  class ProtocolError < StandardError; end

  # Implementation of the server
  class Server
    attr_reader :port

    def initialize(options = {})
      @options = {
        logger:    nil,
        transport: 'tls'
      }.merge!(options)

      @logger = @options[:logger]

      case @options[:transport]
      when 'tcp', 'tls'
        require 'log-courier/server_tcp'
        @server = ServerTcp.new(@options)
      when 'plainzmq', 'zmq'
        require 'log-courier/server_zmq'
        @server = ServerZmq.new(@options)
      else
        raise '[LogCourierServer] \'transport\' must be tcp, tls, plainzmq or zmq'
      end

      # Grab the port back
      @port = @server.port

      # Load the json adapter
      @json_adapter = MultiJson.adapter.instance
    end

    def run(&block)
      # TODO: Make queue size configurable
      event_queue = EventQueue.new 10
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
              @logger.warn("[LogCourierServer] Unknown message received from #{comm.peer}") unless @logger.nil?
              # Don't kill a client that sends a bad message
              # Just reject it and let it send it again, potentially to another server
              comm.send '????', ''
            end
          end
        end

        loop do
          events = event_queue.pop
          events.each do |event|
            block.call event
          end
        end
      ensure
        # Signal the server thread to stop
        unless server_thread.nil?
          server_thread.raise ShutdownSignal
          server_thread.join
        end
      end
    end

    def process_ping(message, comm)
      # Size of message should be 0
      if message.length != 0
        # TODO: log something
        raise ProtocolError
      end

      # PONG!
      # NOTE: comm.send can raise a Timeout::Error of its own
      comm.send 'PONG', ''
    end

    def process_jdat(message, comm, event_queue)
      # Now we have the data, aim to respond within 5 seconds
      reset_ack_timeout

      # OK - first is a nonce - we send this back with sequence acks
      # This allows the client to know what is being acknowledged
      # Nonce is 16 so check we have enough
      if message.length < 17
        # TODO: log something
        raise ProtocolError
      end

      nonce = message[0...16]

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
          @logger.warn("length extraction failed #{ret} #{length_buf.length}")
          # TODO: log something
          raise ProtocolError
        end

        length = length_buf.unpack('N').first

        # Extract message
        ret = message.read length, data_buf
        if ret.nil? or data_buf.length < length
          @logger.warn("message extraction failed #{ret} #{data_buf.length}")
          # TODO: log something
          raise ProtocolError
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
          event = @json_adapter.load(data_buf)
        rescue MultiJson::ParserError => e
          @logger.warn("[LogCourierServer] JSON parse failure, falling back to plain-text: #{e}") unless @logger.nil?
          event = { 'message' => data_buf }
        end

        events << event

        sequence += 1
      end

      # Queue the events
      begin
        event_queue.push events, @ack_timeout - Time.now.to_i
      rescue TimeoutError
        # Full pipeline, partial ack
        # NOTE: comm.send can raise a Timeout::Error of its own
        comm.send('ACKN', [nonce, sequence].pack('A*N'))
        reset_ack_timeout
        retry
      end

      # Acknowledge the full message
      # NOTE: comm.send can raise a Timeout::Error
      comm.send('ACKN', [nonce, sequence].pack('A*N'))
    end

    def reset_ack_timeout
      # TODO: Make a constant or configurable
      @ack_timeout = Time.now.to_i + 5
    end
  end
end
