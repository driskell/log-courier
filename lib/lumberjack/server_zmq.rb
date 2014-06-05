require "ffi-rzmq"
require "timeout"

module Lumberjack
  class ServerZmq
    class ZMQError < StandardError; end

    attr_reader :port

    def initialize(options={})
      @options = {
        :logger  => nil,
        :port    => 0,
        :address => "0.0.0.0",
      }.merge(options)

      @logger = @options[:logger]

      @context = ZMQ::Context.new
      @socket = @context.socket(ZMQ::REP)

      rc = @socket.bind("tcp://" + @options[:address] + (@options[:port] == 0 ? ":*" : ":" + @options[:port].to_s))
      raise ZMQError if !ZMQ::Util.resultcode_ok?(rc)

      # Lookup port number that was allocated in case it was set to 0
      endpoint = ""
      rc = @socket.getsockopt(ZMQ::LAST_ENDPOINT, endpoint)
      raise ZMQError if !ZMQ::Util.resultcode_ok?(rc) or not %r{\Atcp://(?:.*):(?<endpoint_port>\d+)\0\z} =~ endpoint
      @port = Integer(endpoint_port)

      # TODO: Implement workers option by receiving on a ROUTER and proxying to a DEALER, with workers connecting to the DEALER

      reset_timeout
    end

    def run(&block)
      while true
        begin
          # If we don't receive anything after the main timeout - something is probably wrong
          data = Timeout::timeout(@timeout - Time.now.to_i) do
            data = ""
            rc = @socket.recv_string(data)
            raise ZMQError if !ZMQ::Util.resultcode_ok?(rc)
            data
          end
        rescue ZMQError => e
          @logger.warn "[LumberjackServerZMQ] ZMQ recv_string failed: #{e}" if not @logger.nil?
        rescue Timeout::Error
          # We'll let ZeroMQ manage reconnections and new connections
          # There is no point in us doing any form of reconnect ourselves
          # We will keep this timeout in however, for shutdown checks
          reset_timeout
          next
        end
        # We only work with one part messages at the moment
        if @socket.more_parts?
          @logger.warn "[LumberjackServerZMQ] Invalid message: multipart unexpected" if not @logger.nil?
        else
          recv(data, &block)
        end
      end
    rescue ShutdownSignal
      # Shutting down
      @logger.warn("[LumberjackServerZMQ] Server shutting down") if not @logger.nil?
    rescue => e
      # Some other unknown problem
      @logger.warn("[LumberjackServerZMQ] Unknown error: #{e}") if not @logger.nil?
      @logger.debug("[LumberjackServerZMQ] #{e.backtrace}: #{e.message} (#{e.class})") if not @logger.nil? and @logger.debug?
    ensure
      @socket.close
      @context.close
    end

    def recv(data, &block)
      if data.length < 8
        @logger.warn "[LumberjackServerZMQ] Invalid message: not enough data" if not @logger.nil?
        return
      end

      # Unpack the header
      signature, length = data.unpack("A4N")

      # Verify length
      if data.length - 8 != length
        @logger.warn "[LumberjackServerZMQ] Invalid message: data has invalid length (#{data.length - 8} != #{length})" if not @logger.nil?
        return
      end

      # Yield the parts
      yield signature, data[8, length], self
    end

    def send(signature, message)
      reset_timeout
      data = signature + [message.length].pack("N") + message
      Timeout::timeout(@timeout - Time.now.to_i) do
        rc = @socket.send_string(data)
        if !ZMQ::Util.resultcode_ok?(rc)
          @logger.warn "[LumberjackServerZMQ] Message send failed: #{rc}" if not @logger.nil?
          return
        end
      end
    end

    def reset_timeout()
      # TODO: Make configurable?
      @timeout = Time.now.to_i + 1800
    end
  end
end
