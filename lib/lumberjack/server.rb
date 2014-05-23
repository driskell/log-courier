require "socket"
require "thread"
require "timeout"
require "openssl"
require "zlib"
require "json"

module Lumberjack
  class ShutdownSignal < StandardError; end
  class ProtocolError < StandardError; end

  class Server
    attr_reader :port

    # Create a new server with the specified options hash
    #
    # * :port - the port to listen on
    # * :address - the host/address to bind to
    # * :ssl_certificate - the path to the ssl cert to use
    # * :ssl_key - the path to the ssl key to use
    # * :ssl_key_passphrase - the key passphrase (optional)
    # * :ssl_verify - whether to verify client certificates or not
    # * :ssl_verify_default_ca - whether to pass verification on client certificates signed by public authorities
    # * :ssl_verify_ca - the path to the ssl ca to verify client certificates with
    # * :logger - a logger to log to or nil to disable
    def initialize(options={})
      @options = {
        :port => 0,
        :logger => nil,
      }.merge(options)

      @logger = @options[:logger]

      @server = ServerTls.initialise(@options)

      # Grab the port back
      @port = ServerTls.port
    end

    def run(&block)
      # TODO: Make queue size configurable
      event_queue = SizedQueue.new 10
      spooler_thread = nil

      begin
        # Why a spooler thread? Well we don't know what &block is! We want connection threads to be non-blocking so they DON'T timeout
        # Non-blocking means we can keep clients informed of progress, and response in a timely fashion. We could create this with
        # a timeout wrapper around the &block call but we'd then be generating exceptions in someone else's code
        # So we allow the caller to block us - but only our spooler thread - our other threads are safe and we can use timeout
        spooler_thread = Thread.new do
          begin
            while true
              block.call event_queue.pop
            end
          rescue ShutdownSignal
            # Flush whatever we have left
          end
          while event_queue.length
            block.call event_queue.pop
          end
        end

        # Receive messages and process them
        @server.run do |signature, message, comm|
          case signature
            when "PING"
              ping message, comm
            when "JDAT"
              json_data message, comm
            else
              # Don't kill a client that sends a bad message
              # Just reject it and let it send it again, potentially to another server
              comm.send "????", ""
          end
        end
      ensure
        # Signal the spooler thread to stop
        if not spooler_thread.nil?
          spooler_thread.raise ShutdownSignal
          spooler_thread.join
        end
      end
    end

    def ping(message, comm)
      # Size of message should be 0
      if message.length != 0
        raise ProtocolError
      end

      # PONG!
      comm.send "PONG", ""
    end

    def json_data(message, comm)
      # Now we have the data, aim to respond within 5 seconds
      reset_ack_timeout

      # OK - first is a nonce - we send this back with sequence acks
      # This allows the client to know what is being acknowledged
      # Nonce is 32 so check we have enough
      if message.length < 33
        raise ProtocolError
      end

      nonce = message[0...32]

      # The remainder of the message is the compressed data block
      message = Zlib::Inflate.inflate(message[32...message.length])

      # Message now contains JSON encoded events
      # They are aligned as [length][event]... so on
      # We acknowledge them by their 1-index position in the stream
      # A 0 sequence acknowledgement means we haven't processed any yet
      p = 0
      sequence = 0
      while p < message.length
        if message.length - p < 8
          raise ProtocolError
        end

        length = message[p...8].unpack("N").first
        p += 8

        # Check length is valid
        if message.length - p < length
          raise ProtocolError
        end

        # Extract message, and force UTF-8 to ensure we don't break anything, replacing invalid sequences
        data = message[p...length].encode("utf-8", "binary", :invalid => :replace, :undef => :replace, :replace => "?").force_encoding("UTF-8")

        # Decode the JSON
        begin
          event = JSON.parse(data)
        rescue JSON::ParserError => e
          @logger.info("[Lumberjack] JSON parse failure. Falling back to plain-text", :error => e, :data => data)
          event = { "message" => data }
        end

        # Queue the event
        while !queue_event(event)
          # Full pipeline, partial ack
          comm.send(["ACKN", nonce, sequence].pack("A*A*N"))
          reset_ack_timeout
        end

        sequence++
      end

      # Acknowledge the full message
      comm.send(["ACKN", nonce, sequence].pack("A*A*N"))
    end

    def queue_event(event)
      Timeout::timeout(@ack_timeout - Time.now.to_i) do
        event_queue << map
      end
      return true
    rescue Timeout::Error
      return false
    end

    def reset_ack_timeout()
      @ack_timeout = Time.now.to_i + 5
      reset_timeout
    end
  end
end
