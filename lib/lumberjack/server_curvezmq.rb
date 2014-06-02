require "ffi-rzmq"
require "timeout"

module Lumberjack
  class ServerZMQ
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

      rc = @socket.bind("tcp://" + @options[:address] + (@options[:port] == 0 ? "*" : ":" + @options[:port].to_s))
      raise ProtocolError if !ZMQ::Util.resultcode_ok?(rc)

      rc = @socket.getsockopt(ZMQ::LAST_ENDPOINT, endpoint)
      raise ProtocolError if !ZMQ::Util.resultcode_ok?(rc) or %r{\Atcp://(?:.*):(?<endpoint_port>\d+)\0\z} =~ endpoint[0]

      @port = Integer(endpoint_port)

      # TODO: Implement workers option

      reset_timeout
    end

    def run(&block)
      while true
        begin
          # If we don't receive anything after the main timeout - something is probably wrong
          message = Timeout::timeout(@timeout - Time.now.to_i) do
            rc = @socket.recv_string(message)
            raise ProtocolError if !ZMQ::Util.resultcode_ok?(rc)
            message
          end
        rescue Timeout::Error
          # We'll let ZeroMQ manage reconnections and new connections
          # There is no point in us doing any form of reconnect ourselves
          # We will keep this timeout in however, for shutdown checks
          reset_timeout
          next
        end
        # We only work with one part messages at the moment
        raise ProtocolError if @socket.more_parts?
        recv(message, &block)
      end
    end

    def recv(message, &block)
      raise ProtocolError if message.length < 8

      # Unpack the header
      signature, length = message.unpack("A4N")

      # Verify length
      raise ProtocolError if message.length - 8 != length

      # Yield the parts
      # TODO: Just return full message
      yield signature, message[8, length], self
    end

    def send(signature, message)
      reset_timeout
      @logger.warn("ConnectionZmq sending #{@peer} #{signature}") if not @logger.nil?
      data = signature + [message.length].pack("N") + message
      Timeout::timeout(@timeout - Time.now.to_i) do
        rc = @socket.send_string(data)
        raise ProtocolError if !ZMQ::Util.resultcode_ok?(rc)
      end
    end

    def reset_timeout()
      # TODO: Make configurable
      @timeout = Time.now.to_i + 1800
    end
  end
end
