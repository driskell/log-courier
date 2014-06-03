require "json"
require "thread"
require "timeout"
require "zlib"

module Lumberjack
  # TODO: Make these shared
  class ClientShutdownSignal < StandardError; end
  class ClientProtocolError < StandardError; end

  class PendingPayload
    attr_accessor :ack_events
    attr_accessor :events
    attr_accessor :nonce
    attr_accessor :data

    attr_accessor :previous
    attr_accessor :next

    def initialize(options={})
      @ack_events = 0

      options.each do |k, v|
        raise ArgumentError if not self.respond_to?(k)
        instance_variable_set "@#{k}", v
      end
    end
  end

  class Client
    def initialize(options={})
      @options = {
        :logger       => nil,
        :spool_size   => 1024,
        :idle_timeout => 5,
      }.merge(options)

      @logger = @options[:logger]

      require "lumberjack/client_tls"
      @client = ClientTls.new(@options)

      @event_queue = SizedQueue.new @options[:spool_size]
      @pending_payloads = Hash.new
      @first_payload = nil
      @last_payload = nil

      # Start the spooler which will collect events into chunks
      @send_ready = false
      @send_mutex = Mutex.new
      @send_cond = ConditionVariable.new
      @spooler_thread = Thread.new do
        run_spooler
      end

      @pending_ping = false

      # Start the IO thread
      @io_control = SizedQueue.new 1
      @io_thread = Thread.new do
        run_io
      end
    end

    def publish(event)
      # Pass the event into the spooler
      @event_queue << event
    end

    def shutdown
      # Raise a shutdown signal in the spooler and wait for it
      @spooler_thread.raise ClientShutdownSignal
      @io_thread.raise ClientShutdownSignal
      @spooler_thread.join
      @io_thread.join
    end

    def run_spooler
      while true
        spooled = []
        next_flush = Time.now.to_i + @options[:idle_timeout]

        # The spooler loop
        begin
          while true
            event = Timeout::timeout(next_flush - Time.now.to_i) do
              @event_queue.pop
            end
            spooled.push(event)
            break if spooled.length >= @options[:spool_size]
          end
        rescue Timeout::Error
          # Hit timeout but no events, keep waiting
          next if spooled.length == 0
        end

        # Pass through to io_control but only if we're ready to send
        @send_mutex.synchronize do
          if not @send_ready
            @send_cond.wait(@send_mutex)
          end
          @send_ready = false
          @io_control << ["E", spooled]
        end
      end
    rescue ClientShutdownSignal
      # Just shutdown
    end

    def run_io
      # TODO: Make keepalive configurable?
      keepalive_timeout = 1800
      keepalive_next = Time.now.to_i + keepalive_timeout

      # TODO: Make pending payload max configurable?
      max_pending_payloads = 100

      retry_payload = nil

      can_send = true

      while true
        # Reconnect loop
        @client.connect @io_control

        # Capture send exceptions
        begin
          # IO loop
          while true
            catch :keepalive do
              begin
                action = Timeout::timeout(keepalive_next) do
                  @io_control.pop
                end

                # Process the action
                case action[0]
                when "S"
                  # If we're flushing through the pending, pick from there
                  if not retry_payload.nil?
                    # Regenerate data if we need to
                    retry_payload.data = buffer_jdat_data(retry_payload.events, retry_payload.nonce) if retry_payload.data == nil

                    # Send and move onto next
                    @client.send "JDAT", retry_payload.data

                    retry_payload = retry_payload.next
                    throw :keepalive
                  end

                  # Ready to send, allow spooler to pass us something
                  @send_mutex.synchronize do
                    @send_ready = true
                    @send_cond.signal
                  end

                  can_send = true
                when "E"
                  # If we have too many pending payloads, pause the IO
                  if @pending_payloads.length + 1 >= max_pending_payloads
                    @client.pause_send
                  end

                  # Received some events - send them
                  send_jdat action[1]

                  # The send action will trigger another "S" if we have more send buffer
                  can_send = false
                when "R"
                  # Received a message
                  signature, message = action[1..2]
                  case signature
                  when "PONG"
                    process_pong message
                  when "ACKN"
                    process_ackn message
                  else
                    # Unknown message - only listener is allowed to respond with a "????" message
                    # TODO: What should we do? Just ignore for now and let timeouts conquer
                  end
                when "F"
                  # Reconnect, an error occurred
                  break
                end
              rescue Timeout::Error
                # Keepalive timeout hit, send a PING unless we were awaiting a PONG
                if @pending_ping
                  # Timed out, break into reconnect
                  raise Timeout::Error
                end

                # Is send full? can_send will be false if so
                if !can_send
                  # We should've started receiving ACK by now so time out
                  raise Timeout::Error
                end

                # Send PING
                send_ping

                # We may have filled send buffer
                can_send = false
              end
            end

            # Reset keepalive timeout
            keepalive_next = Time.now.to_i + keepalive_timeout
          end
        rescue ClientProtocolError
          # Reconnect required due to a protocol error
          @logger.warn("[LumberjackClient] Protocol error, reconnecting") if not @logger.nil?
        rescue Timeout::Error
          # Reconnect due to timeout
          @logger.warn("[LumberjackClient] Timeout occurred, reconnecting") if not @logger.nil?
        rescue ClientShutdownSignal
          # Shutdown, break out
          break
        rescue => e
          # Unknown error occurred
          @logger.warn("[LumberjackClient] Unknown error: #{e}") if not @logger.nil?
          @logger.debug("[LumberjackClient] #{e.backtrace}: #{e.message} (#{e.class})") if not @logger.nil? and @logger.debug?
        end


        # Disconnect and retry payloads
        @client.disconnect
        retry_payload = @first_payload

        # TODO: Make reconnect time configurable?
        sleep 5
      end

      @client.disconnect
    end

    def generate_nonce
      (0...16).map { rand(256).chr }.join("")
    end

    def send_ping
      # Send it
      @client.send "PING", ""
    end

    def send_jdat(events, is_shutdown=false)
      # Generate the JSON payload and compress it
      nonce = generate_nonce
      data = buffer_jdat_data(events, nonce)

      # Save the pending payload
      payload = PendingPayload.new(
        :events => events,
        :nonce  => nonce,
        :data   => data
      )

      @pending_payloads[nonce] = payload

      if @first_payload.nil?
        @first_payload = payload
        @last_payload = payload
      else
        @last_payload.next = payload
        @last_payload = payload
      end

      # Send it
      @client.send "JDAT", payload.data
    end

    def buffer_jdat_data(events, nonce)
      buffer = Zlib::Deflate.new()

      # Write each event in JSON format
      events.each do |event|
        buffer_jdat_data_event(buffer, event)
      end

      # Generate and return the message
      nonce + buffer.flush(Zlib::FINISH)
    end

    def buffer_jdat_data_event(buffer, event)
      json_data = event.to_json

      # Add length and then the data
      buffer << [json_data.length].pack("N") << json_data
    end

    def process_pong(message)
      # Sanity
      if message.length != 0
        # TODO: log something
        raise ClientProtocolError
      end

      # No longer pending a PONG
      @ping_pending = false
    end

    def process_ackn(message)
      # Sanity
      if message.length != 20
        # TODO: log something
        raise ClientProtocolError
      end

      # Grab nonce
      sequence, nonce = message[0...4].unpack("N").first, message[4..-1]

      # Find the payload
      if !@pending_payloads.has_key?(nonce)
        # Don't error here as we may have had timeout issues and resent a payload only to receive ACKN twice
        return
      end
      payload = @pending_payloads[nonce]

      # Full ACK?
      # TODO: protocol error if sequence too large?
      if sequence >= payload.events.length
        if @client.send_paused?
          @client.resume_send
        end

        @pending_payloads.delete nonce
        payload.previous.next = payload.next
      else
        # Partial ACK - only process if something was actually processed
        if sequence > payload.ack_events
          payload.ack_events = sequence
          payload.events = payload.events[0...sequence]
          payload.data = nil
        end
      end
    end
  end
end
