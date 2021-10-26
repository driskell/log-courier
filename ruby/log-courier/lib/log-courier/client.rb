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

  # Describes a pending payload
  class PendingPayload
    attr_accessor :next, :nonce, :events, :last_sequence, :sequence_len, :payload

    def initialize(events, nonce)
      @events = events
      @nonce = nonce

      generate
    end

    def generate
      raise ArgumentError, 'Corrupt payload' if @events.length.zero?

      buffer = Zlib::Deflate.new

      # Write each event in JSON format
      events.each do |event|
        json_data = MultiJson.dump(event)
        # Add length and then the data
        buffer << [json_data.bytesize].pack('N') << json_data
      end

      # Generate and store the payload
      @payload = nonce + buffer.finish
      @last_sequence = 0
      @sequence_len = @events.length
    end

    def ack(sequence)
      return 0, false if sequence <= @last_sequence

      if sequence >= @sequence_len
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
      [lines, false]
    end
  end

  # Implementation of a single client connection
  class Client
    def initialize(options = {})
      @options = {
        logger: nil,
        transport: 'tls',
        spool_size: 1024,
        idle_timeout: 5,
        port: nil,
        addresses: [],
        min_tls_version: 1.2,
        disable_handshake: false,
      }.merge!(options)

      @logger = @options[:logger]

      case @options[:transport]
      when 'tcp', 'tls'
        require 'log-courier/client_tcp'
        @client = ClientTcp.new(@options)
      else
        raise 'output/courier: \'transport\' must be tcp or tls'
      end

      raise 'output/courier: \'addresses\' must contain at least one address' if @options[:addresses].empty?
      raise 'output/courier: \'addresses\' only supports a single address at this time' if @options[:addresses].length > 1

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
      nil
    end

    def shutdown(force = false) # rubocop:disable Style/OptionalBooleanParameter
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
      @pending_payloads.length.zero?
    end

    private

    def run_spooler
      loop do
        spooled = []
        next_flush = Time.now.to_i + @options[:idle_timeout]

        # The spooler loop
        begin
          loop do
            event = @event_queue.pop next_flush - Time.now.to_i

            raise ShutdownSignal if event.nil?

            spooled.push(event)

            break if spooled.length >= @options[:spool_size]
          end
        rescue TimeoutError
          # Hit timeout but no events, keep waiting
          next if spooled.length.zero?
        end

        if spooled.length >= @options[:spool_size]
          @logger&.debug 'Flushing full spool', events: spooled.length
        else
          @logger&.debug 'Flushing spool due to timeout', events: spooled.length
        end

        # Pass through to io_control but only if we're ready to send
        @send_mutex.synchronize do
          @send_cond.wait(@send_mutex) until @send_ready
          @send_ready = false
        end

        @io_control << ['E', spooled]
      end
    rescue ShutdownSignal
      # Shutdown
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
    rescue ShutdownSignal
      # Ensure disconnected
      @client.disconnect
    end

    def run_io_loop
      io_stop = false
      can_send = false

      # IO loop
      loop do
        action = @io_control.pop @timeout - Time.now.to_i

        # Process the action
        case action[0]
        when 'S'
          # If we're flushing through the pending, pick from there
          unless @retry_payload.nil?
            @logger&.debug 'Send is ready, retrying previous payload'

            # Regenerate data if we need to
            @retry_payload.generate if @retry_payload.payload.nil?

            # Send and move onto next
            @client.send 'JDAT', @retry_payload.payload

            @retry_payload = @retry_payload.next

            # If first send, exit idle mode
            @timeout = Time.now.to_i + @network_timeout if @retry_payload == @first_payload

            next
          end

          # Ready to send, allow spooler to pass us something if we don't
          # have something already
          if @received_payloads.length.zero?
            @logger&.debug 'Send is ready, requesting events'

            can_send = true

            @send_mutex.synchronize do
              @send_ready = true
              @send_cond.signal
            end
          else
            @logger&.debug 'Send is ready, using events from backlog'
            send_payload @received_payloads.pop
          end
        when 'E'
          # Were we expecting a payload? Store it if not
          if can_send
            @logger&.debug 'Sending events', events: action[1].length
            send_payload action[1]
            can_send = false
          else
            @logger&.debug 'Events received when not ready; saved to backlog'
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
          end
          # else
          # Unknown message - only listener is allowed to respond with a "????" message
          # TODO: What should we do? Just ignore for now and let timeouts conquer

          # Any pending payloads left?
          if @pending_payloads.length.zero?
            # Handle shutdown
            raise ShutdownSignal if io_stop

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
          @logger&.debug 'Shutdown request received'

          # Shutdown request received
          raise ShutdownSignal if @pending_payloads.length.zero?

          @logger&.debug 'Delaying shutdown due to pending payloads', payloads: @pending_payloads.length

          io_stop = true

          # Stop spooler sending
          can_send = false
          @send_mutex.synchronize do
            @send_ready = false
          end
        end
      rescue TimeoutError
        # Handle network timeout
        raise TimeoutError if @pending_payloads != 0

        # Keepalive timeout hit, timeout if we were waiting for a pong
        raise TimeoutError if @pending_ping

        # Stop spooler sending
        can_send = false
        @send_mutex.synchronize do
          @send_ready = false
        end

        # Send PING
        send_ping

        @timeout = Time.now.to_i + @network_timeout

        next
      end
    rescue ProtocolError => e
      # Reconnect required due to a protocol error
      @logger&.warn 'Protocol error', error: e.message
    rescue TimeoutError
      # Reconnect due to timeout
      @logger&.warn 'Timeout occurred'
    rescue ShutdownSignal
      raise
    rescue StandardError => e
      # Unknown error occurred
      @logger&.warn e, hint: 'Unknown error'
    end

    def send_payload(payload)
      # If we have too many pending payloads, pause the IO
      @client.pause_send if @pending_payloads.length + 1 >= @max_pending_payloads

      # Received some events - send them
      send_jdat payload

      # Leave idle mode if this is the first payload after idle
      @timeout = Time.now.to_i + @network_timeout if @pending_payloads.length == 1
    end

    def generate_nonce
      (0...16).map { rand(256).chr }.join
    end

    def send_ping
      # Send it
      @client.send 'PING', ''
    end

    def send_jdat(events)
      # Generate the JSON payload and compress it
      nonce = generate_nonce

      # Save the pending payload
      payload = PendingPayload.new(events, nonce)

      @pending_payloads[nonce] = payload

      if @first_payload.nil?
        @first_payload = payload
      else
        @last_payload.next = payload
      end
      @last_payload = payload

      # Send it
      @client.send 'JDAT', payload.payload
    end

    def process_pong(message)
      # Sanity
      raise ProtocolError, "Unexpected data attached to pong message (#{message.bytesize})" if message.bytesize != 0

      @logger&.debug 'PONG message received' || !@logger&.debug?

      # No longer pending a PONG
      @ping_pending = false
    end

    def process_ackn(message)
      # Sanity
      raise ProtocolError, "ACKN message size invalid (#{message.bytesize})" if message.bytesize != 20

      # Grab nonce
      nonce, sequence = message.unpack('a16N')

      if @logger&.debug?
        nonce_str = nonce.each_byte.map do |b|
          b.to_s(16).rjust(2, '0')
        end

        @logger&.debug 'ACKN message received', nonce: nonce_str.join, sequence: sequence
      end

      # Find the payload - skip if we couldn't as it will just a duplicated ACK
      return unless @pending_payloads.key?(nonce)

      payload = @pending_payloads[nonce]
      _, complete = payload.ack(sequence)

      return unless complete

      @pending_payloads.delete nonce
      @first_payload = payload.next
      @client.resume_send if @client.send_paused?
    end
  end
end
