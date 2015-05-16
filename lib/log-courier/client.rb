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
require 'multi_json'
require 'thread'
require 'zlib'

class NativeException; end

module LogCourier
  class TimeoutError < StandardError; end
  class ShutdownSignal < StandardError; end
  class ProtocolError < StandardError; end

  # Describes a pending payload
  class PendingPayload
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

    attr_accessor :next
    attr_accessor :nonce
    attr_accessor :events
    attr_accessor :last_sequence
    attr_accessor :sequence_len
    attr_accessor :payload

    def initialize(events, nonce)
      @events = events
      @nonce = nonce

      generate
    end

    def generate
      fail ArgumentError, 'Corrupt payload' if @events.length == 0

      buffer = Zlib::Deflate.new

      # Write each event in JSON format
      events.each do |event|
        json_data = self.class.get_json_adapter.dump(event)
        # Add length and then the data
        buffer << [json_data.length].pack('N') << json_data
      end

      # Generate and store the payload
      @payload = nonce + buffer.finish()
      @last_sequence = 0
      @sequence_len = @events.length
    end

    def ack(sequence)
      if sequence <= @last_sequence
        return 0, false
      elsif sequence >= @sequence_len
        lines = @sequence_len - @last_sequence
        @last_sequence = sequence
        @payload = nil
        @events = []
        return lines, true
      end

      lines = sequence - @last_sequence
      @last_sequence = sequence
      @payload = nil
      @events.shift(lines)
      return lines, false
    end
  end

  # Implementation of a single client connection
  class Client
    def initialize(options = {})
      @options = {
        transport:    'tls',
        spool_size:   1024,
        idle_timeout: 5,
        port:         nil,
        addresses:    [],
      }.merge!(options)

      if @options[:logger]
        @logger = @options[:logger]
      else
        @logger = Cabin::Channel.new
      end

      case @options[:transport]
      when 'tcp', 'tls'
        require 'log-courier/client_tcp'
        @client = ClientTcp.new(@options)
      else
        fail 'output/courier: \'transport\' must be tcp or tls'
      end

      fail 'output/courier: \'addresses\' must contain at least one address' if @options[:addresses].empty?
      fail 'output/courier: \'addresses\' only supports a single address at this time' if @options[:addresses].length > 1

      @event_queue = EventQueue.new @options[:spool_size]
      @pending_payloads = {}
      @first_payload = nil
      @last_payload = nil

      # Start the spooler which will collect events into chunks
      @send_ready = false
      @send_mutex = Mutex.new
      @send_cond = ConditionVariable.new
      @spooler_thread = Thread.new do
        run_spooler
      end

      # TODO: Make these configurable?
      @keepalive_timeout = 1800
      @network_timeout = 30

      # TODO: Make pending payload max configurable?
      @max_pending_payloads = 100

      @retry_payload = nil
      @received_payloads = Queue.new

      @pending_ping = false

      # Start the IO thread
      @io_control = EventQueue.new 1
      @io_thread = Thread.new do
        run_io
      end
    end

    def publish(event)
      # Pass the event into the spooler
      @event_queue << event
      return
    end

    def shutdown(force=false)
      if force
        # Raise a shutdown signal in the spooler and wait for it
        @spooler_thread.raise ShutdownSignal
        @spooler_thread.join
        @io_thread.raise ShutdownSignal
      else
        @event_queue.push nil
        @spooler_thread.join
        @io_control << ['!', nil]
      end
      @io_thread.join
      return @pending_payloads.length == 0
    end

    private

    def run_spooler
      loop do
        spooled = []
        next_flush = Time.now.to_i + @options[:idle_timeout]

        # The spooler loop
        begin
          loop do
            event = @event_queue.pop [0, next_flush - Time.now.to_i].max

            if event.nil?
              raise ShutdownSignal
            end

            spooled.push(event)

            break if spooled.length >= @options[:spool_size]
          end
        rescue TimeoutError
          # Hit timeout but no events, keep waiting
          next if spooled.length == 0
        end

        if spooled.length >= @options[:spool_size]
          @logger.debug 'Flushing full spool', :events => spooled.length unless @logger.nil?
        else
          @logger.debug 'Flushing spool due to timeout', :events => spooled.length unless @logger.nil?
        end

        # Pass through to io_control but only if we're ready to send
        @send_mutex.synchronize do
          @send_cond.wait(@send_mutex) until @send_ready
          @send_ready = false
        end

        @io_control << ['E', spooled]
      end
      return
    rescue ShutdownSignal
      return
    end

    def run_io
      loop do
        # Reconnect loop
        @client.connect @io_control

        @timeout = Time.now.to_i + @keepalive_timeout

        run_io_loop

        # Disconnect and retry payloads
        @client.disconnect
        @retry_payload = @first_payload

        # TODO: Make reconnect time configurable?
        sleep 5
      end

      @client.disconnect
      return
    rescue ShutdownSignal
      # Ensure disconnected
      @client.disconnect
    end

    def run_io_loop()
      io_stop = false
      can_send = false

      # IO loop
      loop do
        begin
          action = @io_control.pop @timeout - Time.now.to_i

          # Process the action
          case action[0]
          when 'S'
            # If we're flushing through the pending, pick from there
            unless @retry_payload.nil?
              @logger.debug 'Send is ready, retrying previous payload' unless @logger.nil?

              # Regenerate data if we need to
              @retry_payload.generate if @retry_payload.payload.nil?

              # Send and move onto next
              @client.send 'JDAT', @retry_payload.payload

              @retry_payload = @retry_payload.next

              # If first send, exit idle mode
              if @retry_payload == @first_payload
                @timeout = Time.now.to_i + @network_timeout
              end
              next
            end

            # Ready to send, allow spooler to pass us something if we don't
            # have something already
            if @received_payloads.length != 0
              @logger.debug 'Send is ready, using events from backlog' unless @logger.nil?
              send_payload @received_payloads.pop()
            else
              @logger.debug 'Send is ready, requesting events' unless @logger.nil?

              can_send = true

              @send_mutex.synchronize do
                @send_ready = true
                @send_cond.signal
              end
            end
          when 'E'
            # Were we expecting a payload? Store it if not
            if can_send
              @logger.debug 'Sending events', :events => action[1].length unless @logger.nil?
              send_payload action[1]
              can_send = false
            else
              @logger.debug 'Events received when not ready; saved to backlog' unless @logger.nil?
              @received_payloads.push action[1]
            end
          when 'R'
            # Received a message
            signature, message = action[1..2]
            case signature
            when 'PONG'
              process_pong message
            when 'ACKN'
              process_ackn message
            else
              # Unknown message - only listener is allowed to respond with a "????" message
              # TODO: What should we do? Just ignore for now and let timeouts conquer
            end

            # Any pending payloads left?
            if @pending_payloads.length == 0
              # Handle shutdown
              if io_stop
                raise ShutdownSignal
              end

              # Enter idle mode
              @timeout = Time.now.to_i + @keepalive_timeout
            else
              # Set network timeout
              @timeout = Time.now.to_i + @network_timeout
            end
          when 'F'
            # Reconnect, an error occurred
            break
          when '!'
            @logger.debug 'Shutdown request received' unless @logger.nil?

            # Shutdown request received
            if @pending_payloads.length == 0
              raise ShutdownSignal
            end

            @logger.debug 'Delaying shutdown due to pending payloads', :payloads => @pending_payloads.length unless @logger.nil?
            @pending_payloads.each_key do |nonce|
              @logger.debug 'Pending payload', :nonce => nonce_str(nonce)
            end

            io_stop = true

            # Stop spooler sending
            can_send = false
            @send_mutex.synchronize do
              @send_ready = false
            end
          end
        rescue TimeoutError
          if @pending_payloads != 0
            # Network timeout
            fail TimeoutError
          end

          # Keepalive timeout hit, send a PING unless we were awaiting a PONG
          if @pending_ping
            # Timed out, break into reconnect
            fail TimeoutError
          end

          # Stop spooler sending
          can_send = false
          @send_mutex.synchronize do
            @send_ready = false
          end

          # Send PING
          send_ping

          @timeout = Time.now.to_i + @network_timeout
        end
      end
    rescue ProtocolError => e
      # Reconnect required due to a protocol error
      @logger.warn 'Protocol error', :error => e.message unless @logger.nil?
    rescue TimeoutError
      # Reconnect due to timeout
      @logger.warn 'Timeout occurred' unless @logger.nil?
    rescue ShutdownSignal => e
      raise
    rescue StandardError, NativeException => e
      # Unknown error occurred
      @logger.warn e, :hint => 'Unknown error' unless @logger.nil?
    end

    def send_payload(payload)
      # If we have too many pending payloads, pause the IO
      if @pending_payloads.length + 1 >= @max_pending_payloads
        @client.pause_send
      end

      # Received some events - send them
      send_jdat payload

      # Leave idle mode if this is the first payload after idle
      if @pending_payloads.length == 1
        @timeout = Time.now.to_i + @network_timeout
      end
    end

    def generate_nonce
      (0...16).map { rand(256).chr }.join("")
    end

    def send_ping
      # Send it
      @client.send 'PING', ''
      return
    end

    def send_jdat(events)
      # Generate the JSON payload and compress it
      nonce = generate_nonce

      # Save the pending payload
      payload = PendingPayload.new(events, nonce)

      @pending_payloads[nonce] = payload

      if @first_payload.nil?
        @first_payload = payload
        @last_payload = payload
      else
        @last_payload.next = payload
        @last_payload = payload
      end

      @logger.debug 'Sending payload', :nonce => nonce_str(nonce) if !@logger.nil? && @logger.debug?

      # Send it
      @client.send 'JDAT', payload.payload
      return
    end

    def process_pong(message)
      # Sanity
      fail ProtocolError, "Unexpected data attached to pong message (#{message.length})" if message.length != 0

      @logger.debug 'PONG message received' unless @logger.nil? || !@logger.debug?

      # No longer pending a PONG
      @ping_pending = false
      return
    end

    def process_ackn(message)
      # Sanity
      fail ProtocolError, "ACKN message size invalid (#{message.length})" if message.length != 20

      # Grab nonce
      nonce, sequence = message.unpack('a16N')

      @logger.debug 'ACKN message received', :nonce => nonce_str(nonce), :sequence => sequence if !@logger.nil? && @logger.debug?

      # Find the payload - skip if we couldn't as it will just a duplicated ACK
      return unless @pending_payloads.key?(nonce)

      payload = @pending_payloads[nonce]
      lines, complete = payload.ack(sequence)

      if complete
        @pending_payloads.delete nonce
        @first_payload = payload.next
        @client.resume_send if @client.send_paused?
      end
    end

    def nonce_str(nonce)
      nonce.each_byte.map do |b|
        b.to_s(16).rjust(2, '0')
      end.join
    end
  end
end
